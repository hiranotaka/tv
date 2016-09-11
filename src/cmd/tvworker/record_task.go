package main

import (
	"bufio"
	"errors"
	"log"
	"os/exec"
	"strconv"
	"time"
	"zng.jp/tv"
)

type RecordTask struct {
	Event *tv.Event
}

func (task *RecordTask) getFile() string {
	return "/tmp/" + task.Event.Info.Name + ".mp4"
}

func (task *RecordTask) Run(cancel <-chan struct{}) error {
	url, err := task.Event.Program.Stream.Url()
	if err != nil {
		return err
	}

	cmd := exec.Command("env", "LANG=C", "vlc", "-I", "rc", "--sout", "#standard{access=file,dst="+task.getFile(), ",mux=ts}", "--no-sout-all", "--programs", strconv.FormatInt(int64(task.Event.Program.Info.Number), 10), url)
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("StdinPipe failed: %v", err)
	}

	bin := bufio.NewWriter(in)
	cmd.Start()

	waitDone := make(chan struct{})
	go func() {
		cmd.Wait()
		close(waitDone)
	}()

	log.Print("VLC running...")

	select {
	case <-waitDone:
		return errors.New("VLC terminated")
	case <-cancel:
		log.Print("Terminating VLC...")
		timer := time.AfterFunc(time.Second, func() {
			cmd.Process.Kill()
		})
		defer timer.Stop()

		bin.WriteString("quit\n")
		<-waitDone
		return nil
	}
}
