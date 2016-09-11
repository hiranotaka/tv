package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"zng.jp/tv"
)

func isRecordTaskForEvent(task Task, event *tv.Event) bool {
	if recordTask, ok := task.(*RecordTask); ok {
		return recordTask.Event.Program.Info.Number == event.Program.Info.Number && recordTask.Event.Info.Name == event.Info.Name
	} else {
		return false
	}
}

func isScanTask(task Task) bool {
	_, ok := task.(*ScanTask)
	return ok
}

type job struct {
	task Task
	end  time.Time
}

func pickNextJob(data *tv.Data, currentJob *job, now time.Time) *job {
	idleEnd := now.Add(24 * time.Hour)

	if event := data.CurrentMatchedEvent(now); event != nil {
		if currentJob != nil && isRecordTaskForEvent(currentJob.task, event) {
			return &job{
				task: currentJob.task,
				end:  event.End(),
			}
		} else {
			return &job{
				task: &RecordTask{Event: event},
				end:  event.End(),
			}
		}
	}

	if event := data.NextMatchedEvent(now); event != nil {
		idleEnd = event.Info.Start
	}

	if currentJob != nil && isScanTask(currentJob.task) {
		if idleEnd.Before(currentJob.end) {
			return &job{
				task: currentJob.task,
				end:  idleEnd,
			}
		} else {
			return currentJob
		}
	}

	stream := data.StreamWithoutInfoOrWithOldestInfo()
	if stream != nil {
		var scanStart time.Time
		if stream.Info != nil {
			scanStart = stream.Info.Time.Add(6 * time.Hour)
		}
		if scanStart.Before(now) {
			if idleEnd.Sub(now) > 120*time.Second {
				return &job{
					task: &ScanTask{Stream: stream},
					end:  now.Add(120 * time.Second),
				}
			}
		} else if idleEnd.Sub(scanStart) > 120*time.Second {
			idleEnd = scanStart
		}
	}

	return &job{
		task: &IdleTask{},
		end:  idleEnd,
	}
}

func fetchData() (*tv.Data, error) {
	log.Print("Fetching data...")
	response, err := http.Get("http://zng.jp/tv/tvctl.cgi?mode=json")
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return nil, errors.New("Server returned on-OK status: " + strconv.Itoa(response.StatusCode) + " " + string(body))
	}

	data := &tv.Data{}
	if err := json.NewDecoder(response.Body).Decode(data); err != nil {
		return nil, err
	}

	data.MergeData(tv.StreamConfigData)

	return data, nil
}

func listen(receiveDone chan<- struct{}) error {
	response, err := http.Get("http://zng.jp/tv/tvctl.cgi?mode=event-stream")
	if err != nil {
		return err
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return errors.New("Server returned on-OK status: " + strconv.Itoa(response.StatusCode) + " " + string(body))
	}

	scanner := bufio.NewScanner(response.Body)
	data := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			data = strings.TrimSuffix(data, "\n")
			if data != "" {
				receiveDone <- struct{}{}
				data = ""
			}
		} else {
			fieldValue := strings.SplitN(line, ":", 2)
			field := fieldValue[0]
			value := ""
			if len(fieldValue) == 2 {
				value = fieldValue[1]
			}
			value = strings.TrimPrefix(value, " ")

			if field == "data" {
				data += value
			}
		}
	}

	return scanner.Err()
}

func main() {
	receiveDone := make(chan struct{})
	go func() {
		for {
			err := listen(receiveDone)
			log.Printf("Listen failed: %v", err)
			time.Sleep(30 * time.Second)
		}
	}()

	data := &tv.Data{}
	var job *job
	var runCancel chan struct{}
	var runDone chan struct{}
	var runErr error

	for {
		nextJob := pickNextJob(data, job, time.Now())
		if job != nil && job.task != nextJob.task {
			log.Print("Terminating task...")
			close(runCancel)
			<-runDone
		}
		if job == nil || job.task != nextJob.task {
			runCancel = make(chan struct{})
			runDone = make(chan struct{})
			go func() {
				runErr = nextJob.task.Run(runCancel)
				close(runDone)
			}()

			job = nextJob
		}

		timer := time.NewTimer(job.end.Sub(time.Now()))
		defer timer.Stop()

		select {
		case <-runDone:
			job = nil
		case <-timer.C:
			log.Print("Terminating task...")
			close(runCancel)
			<-runDone

			job = nil
		case <-receiveDone:
			newData, err := fetchData()
			if err != nil {
				log.Printf("fetchData failed: %v", err)
				break
			}
			data = newData
		}
	}
}
