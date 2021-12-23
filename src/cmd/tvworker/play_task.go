package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os/exec"
	"strconv"
	"time"
	"zng.jp/tv"
)

type PlayTask struct {
	Program *tv.Program
	Writer  io.Writer
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

func (task *PlayTask) Run(cancel <-chan struct{}, assignments []int32) error {
	url, err := task.Program.Stream.Url(assignments[0])
	if err != nil {
		return err
	}

	cmd := exec.Command("env", "LANG=C", "vlc", "-I", "rc", "--sout", "#standard{access=file,dst=-,mux=ts}", "--no-sout-all", "--programs", strconv.FormatInt(int64(task.Program.Info.Number), 10), url)
	cmd.Stdout = task.Writer
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
