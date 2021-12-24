package main

import (
	"bufio"
	"fmt"
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
	return "/srv/tv/" + task.Event.Info.Name + ".ts"
}

func (task *RecordTask) String() string {
	return fmt.Sprintf("RecordTask{%v %v}", task.Event.Program.Info.Number, task.Event.Info.Name)
}

func (task *RecordTask) Requirements() []int32 {
	return []int32{task.Event.Program.Stream.Config.System}
}

func (task *RecordTask) Equals(otherTask Task) bool {
	otherRecordTask, ok := otherTask.(*RecordTask)
	if !ok {
		return false
	}
	return otherRecordTask.Event.Program.Info.Number == task.Event.Program.Info.Number && otherRecordTask.Event.Info.Name == task.Event.Info.Name
}

func (task *RecordTask) Run(cancel <-chan struct{}, assignments []int32) {
	url, err := task.Event.Program.Stream.Url(assignments[0])
	if err != nil {
		log.Print("Failed to get a URL")
		return
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

	select {
	case <-waitDone:
		log.Print("VLC terminated")
	case <-cancel:
		timer := time.AfterFunc(time.Second, func() {
			log.Print("VLC is not terminating within a second.")
			cmd.Process.Kill()
		})
		defer timer.Stop()

		bin.WriteString("quit\n")
		<-waitDone
	}
}
