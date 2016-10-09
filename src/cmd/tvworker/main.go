package main

import (
	"log"
	"time"
	"zng.jp/tv"
	"zng.jp/tv/db"
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

	stream := data.StreamWithoutStateOrWithOldestState()
	if stream != nil {
		var scanStart time.Time
		if stream.State != nil {
			scanStart = stream.State.Time.Add(3 * time.Hour)
		}
		if scanStart.Before(now) {
			if idleEnd.Sub(now) > 305*time.Second {
				return &job{
					task: &ScanTask{Time: now, Stream: stream},
					end:  now.Add(305 * time.Second),
				}
			}
		} else if idleEnd.Sub(scanStart) > 305*time.Second {
			idleEnd = scanStart
		}
	}

	return &job{
		task: &IdleTask{},
		end:  idleEnd,
	}
}

func main() {
	notificationQueue := make(chan struct{})
	go func() {
		defer close(notificationQueue)
		for {
			err := db.ListenData(notificationQueue)
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
		}
		job = nextJob

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
		case <-notificationQueue:
			log.Print("Fetching data...")
			newData, err := db.FetchData()
			if err != nil {
				log.Printf("fetchData failed: %v", err)
				break
			}
			data = newData
		}
	}
}
