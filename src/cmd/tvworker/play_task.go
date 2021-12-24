package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"time"
	"zng.jp/tv"
)

type PlayTask struct {
	Program *tv.Program
	Writer  chan io.Writer
}

func (task *PlayTask) String() string {
	return fmt.Sprintf("PlayTask{%v}", task.Writer)
}

func (task *PlayTask) Requirements() []int32 {
	return []int32{task.Program.Stream.Config.System}
}

func (task *PlayTask) Equals(otherTask Task) bool {
	otherPlayTask, ok := otherTask.(*PlayTask)
	if !ok {
		return false
	}
	return otherPlayTask.Writer == task.Writer
}

func (task *PlayTask) Run(cancel <-chan struct{}, assignments []int32) {
	url, err := task.Program.Stream.Url(assignments[0])
	if err != nil {
		log.Print("Failed to get a URL")
		return
	}

	writer, ok := <-task.Writer
	if !ok {
		log.Print("Failed to acquire a writer")
		return
	}
	defer func() { task.Writer <- writer }()

	cmd := exec.Command("env", "LANG=C", "vlc", "-I", "rc", "--sout", "#standard{access=file,dst=-,mux=ts}", "--no-sout-all", "--programs", strconv.FormatInt(int64(task.Program.Info.Number), 10), url)
	cmd.Stdout = writer
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("StdinPipe failed: %v", err)
	}
	cmd.Start()

	waitDone := make(chan struct{})
	go func() {
		cmd.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		log.Print("VLC terminated")
	case <-cancel:
		timer := time.AfterFunc(time.Second, func() {
			log.Print("VLC is not terminating within a second.")
			cmd.Process.Kill()
		})
		defer timer.Stop()

		io.WriteString(in, "quit\n")
		<-waitDone
	}
}
