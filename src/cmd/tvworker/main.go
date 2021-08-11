package main

import (
	"log"
	"sort"
	"time"
	"zng.jp/tv"
	"zng.jp/tv/db"
)

type job struct {
	task        Task
	assignments []int32
	canceling   bool
	cancel      chan struct{}
}

type minTimeTracker struct {
	time time.Time
}

func (tracker *minTimeTracker) Update(t time.Time) {
	if t.Before(tracker.time) {
		tracker.time = t
	}
}

type streamsByStateTime []*tv.Stream

func (streams streamsByStateTime) Len() int {
	return len(streams)
}
func (streams streamsByStateTime) Swap(i, j int) {
	streams[i], streams[j] = streams[j], streams[i]
}
func (streams streamsByStateTime) Less(i, j int) bool {
	if streams[i].State == nil || streams[j].State == nil {
		return streams[j].State != nil
	} else {
		return streams[i].State.Time.Before(streams[j].State.Time)
	}
}

type scheduler struct {
	tasks     []Task
	resources map[int32]int
}

func (s *scheduler) MaybeAdd(task Task) {
	if s.resources == nil {
		s.resources = map[int32]int{
			tv.ISDB_T: 2,
			tv.ISDB_S: 2,
		}
	}
	for _, requirement := range task.Requirements() {
		if s.resources[requirement] <= 0 {
			return
		}
	}
	for _, requirement := range task.Requirements() {
		s.resources[requirement]--
	}
	s.tasks = append(s.tasks, task)
}

func schedule(data *tv.Data, now time.Time) ([]Task, time.Time) {
	nextTime := minTimeTracker{time: now.Add(24 * time.Hour)}
	scheduler := scheduler{}

	var eventsToRecord []*tv.Event
	for _, event := range data.Events() {
		if event.Info == nil {
			continue
		}
		if data.RuleMatchingEvent(event) == nil {
			continue
		}
		if event.IsCurrent(now) {
			eventsToRecord = append(eventsToRecord, event)
			nextTime.Update(event.End())
		} else if now.Before(event.Info.Start) {
			nextTime.Update(event.Info.Start)
		}
	}
	for _, event := range eventsToRecord {
		scheduler.MaybeAdd(&RecordTask{Event: event})
	}

	var streamsToScan []*tv.Stream
	for _, stream := range data.Streams() {
		var scanStart time.Time
		if stream.State != nil {
			scanStart = stream.State.Time.Add(3 * time.Hour)
		}
		if scanStart.Before(now) {
			streamsToScan = append(streamsToScan, stream)
		} else {
			nextTime.Update(scanStart)
		}
	}
	sort.Sort(streamsByStateTime(streamsToScan))
	for _, stream := range streamsToScan {
		scheduler.MaybeAdd(&ScanTask{Time: now, Stream: stream})
	}

	return scheduler.tasks, nextTime.time
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
	jobs := []*job{}
	resources := map[int32][]int32{
		tv.ISDB_S: []int32{0, 1},
		tv.ISDB_T: []int32{0, 1},
	}
	jobDone := make(chan *job)

	for {
		tasks, nextTime := schedule(data, time.Now())

		for _, job := range jobs {
			shouldRun := false
			for _, task := range tasks {
				if job.task.Equals(task) {
					shouldRun = true
					break
				}
			}

			if shouldRun {
				continue
			}

			if job.canceling {
				continue
			}

			log.Printf("Terminating task...", job.task)
			close(job.cancel)
			job.canceling = true
		}

		for _, task := range tasks {
			running := false
			for _, job := range jobs {
				if task.Equals(job.task) {
					running = true
					break
				}
			}

			if running {
				continue
			}

			runnable := true
			for _, requirement := range task.Requirements() {
				if len(resources[requirement]) <= 0 {
					runnable = false
				}
			}

			if !runnable {
				continue
			}

			assignments := make([]int32, len(task.Requirements()))
			for i, requirement := range task.Requirements() {
				assignments[i] = resources[requirement][len(resources[requirement])-1]
				resources[requirement] = resources[requirement][0 : len(resources[requirement])-1]
			}

			cancel := make(chan struct{})

			job := &job{
				task:        task,
				assignments: assignments,
				cancel:      cancel,
			}

			jobs = append(jobs, job)

			go func() {
				job.task.Run(cancel, assignments)
				jobDone <- job
			}()
		}

		timer := time.NewTimer(nextTime.Sub(time.Now()))

		select {
		case doneJob := <-jobDone:
			timer.Stop()
			for i, job := range jobs {
				if doneJob == job {
					jobs[i] = jobs[len(jobs)-1]
					jobs = jobs[0 : len(jobs)-1]
					break
				}
			}
			for i, requirement := range doneJob.task.Requirements() {
				resources[requirement] = append(resources[requirement], doneJob.assignments[i])
			}

		case <-timer.C:

		case <-notificationQueue:
			timer.Stop()
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
