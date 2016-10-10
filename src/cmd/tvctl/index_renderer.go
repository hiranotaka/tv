package main

import (
	"html/template"
	"io"
	"net/url"
	"sort"
	timepkg "time"
	"zng.jp/tv"
)

var (
	indexTemplate = template.Must(template.ParseFiles("assets/index.tmpl"))
)

type time struct {
	Index           int
	Time            timepkg.Time
	startingEvents  []*tv.Event
	endingEvents    []*tv.Event
	StartingHour    *hour
	EndingHour      *hour
	StartingSlotMap map[tv.ProgramId]*slot
}

type timeInterval struct {
	Start *time
	End   *time
}

type slot struct {
	TimeInterval timeInterval
	Event        *tv.Event
}

type hour struct {
	TimeInterval timeInterval
	Hour         int
}

func (interval *timeInterval) Span() int {
	return interval.End.Index - interval.Start.Index
}

type programsByNumberAsc []*tv.Program

func (programs programsByNumberAsc) Len() int {
	return len(programs)
}

func (programs programsByNumberAsc) Less(i, j int) bool {
	return programs[i].Info.Number < programs[j].Info.Number
}

func (programs programsByNumberAsc) Swap(i, j int) {
	programs[i], programs[j] = programs[j], programs[i]
}

type timesAsc []*time

func (times timesAsc) Len() int {
	return len(times)
}

func (times timesAsc) Less(i, j int) bool {
	return times[i].Time.Before(times[j].Time)
}

func (times timesAsc) Swap(i, j int) {
	times[i], times[j] = times[j], times[i]
}

type indexTemplateArgs struct {
	Data            *tv.Data
	Programs        []*tv.Program
	TimeIntervals   []timeInterval
	Days            []timepkg.Time
	SelectedDay     timepkg.Time
	SelectedEventId tv.EventId
	WantEvent       bool
}

func renderIndex(data *tv.Data, query url.Values, writer io.Writer) error {
	now := timepkg.Now()

	var selectedTime timepkg.Time
	selectedTimeStr := query.Get("time")
	if selectedTimeStr != "" {
		var err error
		selectedTime, err = timepkg.Parse("2006-01-02 15:04:05.999999999 -0700 MST", selectedTimeStr)
		if err != nil {
			return err
		}
	} else {
		selectedTime = now
	}
	selectedEventId := tv.EventId(query.Get("selected-event"))
	wantEvent := query.Get("want-event") != ""

	selectedDayStart := timepkg.Date(selectedTime.Year(), selectedTime.Month(), selectedTime.Day(), 0, 0, 0, 0, selectedTime.Location())
	selectedDayEnd := timepkg.Date(selectedTime.Year(), selectedTime.Month(), selectedTime.Day()+1, 0, 0, 0, 0, selectedTime.Location())

	programs := data.Programs()
	sort.Sort(programsByNumberAsc(programs))

	var times []*time
	for _, event := range data.Events() {
		start := event.Info.Start
		end := event.End()
		if !selectedDayStart.Before(end) || !start.Before(selectedDayEnd) {
			continue
		}

		if start.Before(selectedDayStart) {
			start = selectedDayStart
		}

		if selectedDayEnd.Before(end) {
			end = selectedDayEnd
		}

		times = append(times, &time{
			Time:           start,
			startingEvents: []*tv.Event{event},
		}, &time{
			Time:         end,
			endingEvents: []*tv.Event{event},
		})
	}

	for hourOffset := 0; hourOffset < 24; hourOffset++ {
		start := timepkg.Date(selectedDayStart.Year(), selectedDayStart.Month(), selectedDayStart.Day(), selectedDayStart.Hour()+hourOffset, 0, 0, 0, selectedDayStart.Location())
		end := timepkg.Date(selectedDayStart.Year(), selectedDayStart.Month(), selectedDayStart.Day(), selectedDayStart.Hour()+hourOffset+1, 0, 0, 0, selectedDayStart.Location())
		hour := &hour{Hour: start.Hour()}
		times = append(times, &time{
			Time:         start,
			StartingHour: hour,
		}, &time{
			Time:       end,
			EndingHour: hour,
		})
	}

	sort.Sort(timesAsc(times))

	var uniqueTimes []*time
	var uniqueTime *time
	for _, aTime := range times {
		if uniqueTime == nil || !aTime.Time.Equal(uniqueTime.Time) {
			uniqueTime = &time{Time: aTime.Time, Index: len(uniqueTimes)}
			uniqueTimes = append(uniqueTimes, uniqueTime)
		}
		uniqueTime.startingEvents = append(uniqueTime.startingEvents, aTime.startingEvents...)
		uniqueTime.endingEvents = append(uniqueTime.endingEvents, aTime.endingEvents...)
		if aTime.StartingHour != nil {
			uniqueTime.StartingHour = aTime.StartingHour
		}
		if aTime.EndingHour != nil {
			uniqueTime.EndingHour = aTime.EndingHour
		}
	}

	var timeIntervals []timeInterval
	slotMap := make(map[tv.ProgramId]*slot)
	for _, time := range uniqueTimes {
		time.StartingSlotMap = make(map[tv.ProgramId]*slot)

		if time.Index == 0 {
			for _, program := range data.Programs() {
				time.StartingSlotMap[program.Id()] = &slot{}
			}
		} else {
			for _, event := range time.endingEvents {
				if event == slotMap[event.Program.Id()].Event {
					time.StartingSlotMap[event.Program.Id()] = &slot{}
				}
			}
			timeIntervals[len(timeIntervals)-1].End = time
		}

		if time.Index != len(uniqueTimes)-1 {
			startingTimeInterval := timeInterval{}
			startingTimeInterval.Start = time
			timeIntervals = append(timeIntervals, startingTimeInterval)

			for _, event := range time.startingEvents {
				time.StartingSlotMap[event.Program.Id()] = &slot{Event: event}
			}
		} else {
			for _, program := range data.Programs() {
				time.StartingSlotMap[program.Id()] = nil
			}
		}

		for programId, startingSlot := range time.StartingSlotMap {
			if slotMap[programId] != nil {
				slotMap[programId].TimeInterval.End = time
			}
			if startingSlot != nil {
				startingSlot.TimeInterval.Start = time
				slotMap[programId] = startingSlot
			} else {
				delete(slotMap, programId)
			}
		}
	}

	for _, time := range uniqueTimes {
		if time.StartingHour != nil {
			time.StartingHour.TimeInterval.Start = time
		}
		if time.EndingHour != nil {
			time.EndingHour.TimeInterval.End = time
		}
	}

	var days []timepkg.Time
	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		days = append(days, timepkg.Date(now.Year(), now.Month(), now.Day()+dayOffset, 0, 0, 0, 0, now.Location()))
	}

	args := &indexTemplateArgs{
		Data:            data,
		Programs:        programs,
		TimeIntervals:   timeIntervals,
		SelectedEventId: selectedEventId,
		Days:            days,
		SelectedDay:     selectedDayStart,
		WantEvent:       wantEvent,
	}

	return indexTemplate.Execute(writer, args)
}
