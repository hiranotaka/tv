package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"time"
	"zng.jp/tv"
	"zng.jp/tv/db"
)

type ScanTask struct {
	Time   time.Time
	Stream *tv.Stream
}

func sleep(cancel <-chan struct{}, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	select {
	case <-timer.C:
		return true
	case <-cancel:
		timer.Stop()
		return false
	}
}

type sectionParser interface {
	parseLine(line string) error
}

type programInfoParser struct {
	programInfo *tv.ProgramInfo
}

func (parser *programInfoParser) parseLine(line string) error {
	if matches := regexp.MustCompile(`^\| (\d\d\d\d)-(\d\d)-(\d\d) (\d\d):(\d\d):(\d\d): (.*) \((\d\d):(\d\d)\) - (.*)`).FindStringSubmatch(line); matches != nil {
		startYear, _ := strconv.Atoi(matches[1])
		startMonthInt, _ := strconv.Atoi(matches[2])
		startMonth := time.Month(startMonthInt)
		startDay, _ := strconv.Atoi(matches[3])
		startHour, _ := strconv.Atoi(matches[4])
		startMin, _ := strconv.Atoi(matches[5])
		startSec, _ := strconv.Atoi(matches[6])
		name := matches[7]
		durationHour, _ := strconv.Atoi(matches[8])
		durationMin, _ := strconv.Atoi(matches[9])
		description := matches[10]

		location, err := time.LoadLocation("Asia/Tokyo")
		if err != nil {
			return err
		}

		start := time.Date(startYear, startMonth, startDay, startHour, startMin, startSec, 0, location)
		duration := time.Duration(durationHour)*time.Hour + time.Duration(durationMin)*time.Minute

		parser.programInfo.Events = append(parser.programInfo.Events, &tv.EventInfo{
			Start:       start,
			Duration:    duration,
			Name:        name,
			Description: description,
		})
	}
	return nil
}

func (task *ScanTask) parseStreamInfo(scanner *bufio.Scanner) (*tv.StreamInfo, error) {
	var programsInfo []*tv.ProgramInfo
	programInfoMap := make(map[int32]*tv.ProgramInfo)
	var sectionParser sectionParser
	for scanner.Scan() {
		line := scanner.Text()

		if matches := regexp.MustCompile(`^(?:> )?\+----\[ (.*) \]$`).FindStringSubmatch(line); matches != nil {
			section := matches[1]
			if section == "end of stream info" {
				return &tv.StreamInfo{
					Time:     task.Time,
					Programs: programsInfo,
				}, nil
			}

			sectionParser = nil
			if matches := regexp.MustCompile(`^EPG (.*) \[Program (\d+)\] \[Table (\d+)\]$`).FindStringSubmatch(section); matches != nil {
				serviceName := matches[1]
				programNumber, _ := strconv.ParseInt(matches[2], 10, 32)
				tableId, _ := strconv.ParseInt(matches[3], 10, 32)

				if tableId < 0x50 || tableId >= 0x60 {
					continue
				}

				programInfo := programInfoMap[int32(programNumber)]
				if programInfo == nil {
					programInfo = &tv.ProgramInfo{
						Number: int32(programNumber),
						Title:  serviceName,
					}
					programsInfo = append(programsInfo, programInfo)
					programInfoMap[int32(programNumber)] = programInfo
				}
				sectionParser = &programInfoParser{programInfo: programInfo}
			}
		}

		if sectionParser != nil {
			if err := sectionParser.parseLine(line); err != nil {
				return nil, err
			}
		}
	}
	return nil, scanner.Err()
}

func (task *ScanTask) communicate(cancel <-chan struct{}, in io.Writer, scanner *bufio.Scanner) (*tv.StreamInfo, error) {
	if !sleep(cancel, 300*time.Second) {
		return nil, errors.New("Cancelled")
	}

	log.Print("Retrieving stream info...")
	if _, err := io.WriteString(in, "info\n"); err != nil {
		return nil, err
	}

	streamInfo, err := task.parseStreamInfo(scanner)
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Second)

	log.Print("Terminating VLC...")
	if _, err := io.WriteString(in, "quit\n"); err != nil {
		return nil, err
	}

	return streamInfo, nil
}

func (task *ScanTask) scanStreamInfo(cancel <-chan struct{}) (*tv.StreamInfo, error) {
	url, err := task.Stream.Url()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("env", "LANG=C", "vlc", "-I", "oldrc", "--rc-fake-tty", "--no-audio", "--no-video", url)
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("cmd.StdinPipe failed: %v", err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("cmd.StdoutPipe failed: %v", err)
	}

	scanner := bufio.NewScanner(out)

	cmd.Start()

	communicateCancel := make(chan struct{})
	communicateDone := make(chan struct{})
	var streamInfo *tv.StreamInfo
	var communicateErr error
	go func() {
		streamInfo, communicateErr = task.communicate(communicateCancel, in, scanner)
		close(communicateDone)
	}()

	waitDone := make(chan struct{})
	go func() {
		cmd.Wait()
		close(waitDone)
	}()

	select {
	case <-cancel:
		log.Print("Killing VLC...")
		cmd.Process.Kill()
		close(communicateCancel)
		<-communicateDone
		<-waitDone
		return nil, errors.New("Cancelled")

	case <-communicateDone:
		if communicateErr != nil {
			log.Print("Killing VLC...")
			cmd.Process.Kill()
			<-waitDone
			return nil, err
		}

		timer := time.AfterFunc(time.Second, func() {
			log.Print("Killing VLC...")
			cmd.Process.Kill()
		})
		defer timer.Stop()
		<-waitDone

		return streamInfo, nil

	case <-waitDone:
		log.Print("Cancelling communication...")
		close(communicateCancel)
		<-communicateDone
		if communicateErr != nil {
			return nil, errors.New("VLC terminated")
		}

		return streamInfo, nil
	}
}

func (task *ScanTask) Run(cancel <-chan struct{}) error {
	data := &tv.Data{
		StreamStateMap: map[tv.StreamId]*tv.StreamState{
			task.Stream.Id: &tv.StreamState{
				Time: task.Time,
			},
		},
	}

	log.Printf("Scanning stream info: %s ...", task.Stream.Id)
	streamInfo, err := task.scanStreamInfo(cancel)
	if err == nil {
		data.InsertStreamInfo(task.Stream.Id, streamInfo)
	} else if err.Error() == "Cancelled" {
		return err
	}

	log.Printf("Submitting stream info: %s ...", task.Stream.Id)
	if err := db.PostData(cancel, data); err != nil {
		return err
	}

	return nil
}
