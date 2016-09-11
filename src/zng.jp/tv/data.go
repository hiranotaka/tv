package tv

import (
	"errors"
	"fmt"
	"time"
)

const (
	ISDB_T = 1
	ISDB_S = 2
)

type EventId string
type ProgramId string
type StreamId string
type RuleId string

type StreamConfig struct {
	System    int32
	Frequency int32
	TsId      int32
}

type RuleConfig struct {
	Deleted       bool
	ProgramNumber int32
	Start         time.Time
}

type EventInfo struct {
	Start       time.Time
	Duration    time.Duration
	Name        string
	Description string
}

type ProgramInfo struct {
	Number int32
	Title  string
	Events []*EventInfo
}

type StreamInfo struct {
	Time     time.Time
	Programs []*ProgramInfo
}

type Stream struct {
	Id     StreamId
	Config *StreamConfig
	Info   *StreamInfo
}

type Program struct {
	Info          *ProgramInfo
	Stream        *Stream
	IndexInStream int
}

type Event struct {
	Info           *EventInfo
	Program        *Program
	IndexInProgram int
}

type Rule struct {
	Id     RuleId
	Config *RuleConfig
}

type Data struct {
	StreamMap map[StreamId]*Stream
	RuleMap   map[RuleId]*Rule
}

func (event *Event) Id() EventId {
	return EventId(fmt.Sprintf("%d@%s", event.IndexInProgram, event.Program.Id()))
}

func (event *Event) End() time.Time {
	return event.Info.Start.Add(event.Info.Duration)
}

func (event *Event) IsCurrent(now time.Time) bool {
	return event.Info.Start.Before(now) && now.Before(event.End())
}

func (program *Program) Id() ProgramId {
	return ProgramId(fmt.Sprintf("%d@%s@%s", program.IndexInStream, program.Stream.Info.Time.String(), program.Stream.Id))
}

func (program *Program) Events() (events []*Event) {
	for index, eventInfo := range program.Info.Events {
		events = append(events, &Event{
			Info:           eventInfo,
			Program:        program,
			IndexInProgram: index,
		})
	}
	return
}

func (stream *Stream) Programs() (programs []*Program) {
	for index, programInfo := range stream.Info.Programs {
		programs = append(programs, &Program{
			Info:          programInfo,
			Stream:        stream,
			IndexInStream: index,
		})
	}
	return
}

func (stream *Stream) Url() (string, error) {
	config := stream.Config
	switch config.System {
	case ISDB_T:
		return fmt.Sprintf("isdb-t://adapter=3:frequency=%d", config.Frequency), nil
	case ISDB_S:
		return fmt.Sprintf("isdb-s://adapter=2:frequency=%d:ts-id=%d", config.Frequency, config.TsId), nil
	default:
		return "", errors.New("Unknown system")
	}
}

func (rule *Rule) MatchEvent(event *Event) bool {
	return event.Program.Info.Number == rule.Config.ProgramNumber && event.Info.Start == rule.Config.Start
}

func (data *Data) LookupOrInsertStream(id StreamId) *Stream {
	stream := data.StreamMap[id]
	if stream == nil {
		stream = &Stream{Id: id}
		if data.StreamMap == nil {
			data.StreamMap = make(map[StreamId]*Stream)
		}
		data.StreamMap[id] = stream
	}
	return stream
}

func (data *Data) LookupOrInsertRule(id RuleId) *Rule {
	rule := data.RuleMap[id]
	if rule == nil {
		rule = &Rule{Id: id}
		if data.RuleMap == nil {
			data.RuleMap = make(map[RuleId]*Rule)
		}
		data.RuleMap[id] = rule
	}
	return rule
}

func (data *Data) MergeData(newData *Data) {
	for id, newStream := range newData.StreamMap {
		stream := data.LookupOrInsertStream(id)
		if newStream.Config != nil {
			stream.Config = newStream.Config
		}
		if newStream.Info != nil {
			stream.Info = newStream.Info
		}
	}

	for id, newRule := range newData.RuleMap {
		rule := data.LookupOrInsertRule(id)
		if newRule.Config != nil {
			if !newRule.Config.Deleted {
				rule.Config = newRule.Config
			} else {
				rule.Config = nil
			}
		}
	}
}

func (data *Data) Programs() (programs []*Program) {
	for _, stream := range data.StreamMap {
		if stream.Info == nil {
			continue
		}
		programs = append(programs, stream.Programs()...)
	}
	return
}

func (data *Data) Events() (events []*Event) {
	for _, program := range data.Programs() {
		if program.Info == nil {
			continue
		}
		events = append(events, program.Events()...)
	}
	return
}

func (data *Data) StreamWithoutInfoOrWithOldestInfo() *Stream {
	for _, stream := range data.StreamMap {
		if stream.Info == nil {
			return stream
		}
	}

	var streamWithOldestInfo *Stream
	for _, stream := range data.StreamMap {
		if streamWithOldestInfo == nil || stream.Info.Time.Before(streamWithOldestInfo.Info.Time) {
			streamWithOldestInfo = stream
		}
	}

	return streamWithOldestInfo
}

func (data *Data) RuleMatchingEvent(event *Event) *Rule {
	for _, rule := range data.RuleMap {
		if rule.Config == nil {
			continue
		}
		if rule.MatchEvent(event) {
			return rule
		}
	}
	return nil
}

func (data *Data) CurrentMatchedEvent(now time.Time) *Event {
	for _, event := range data.Events() {
		if event.Info == nil {
			continue
		}
		if event.IsCurrent(now) && data.RuleMatchingEvent(event) != nil {
			return event
		}
	}
	return nil
}

func (data *Data) NextMatchedEvent(now time.Time) (nextEvent *Event) {
	for _, event := range data.Events() {
		if event.Info == nil {
			continue
		}
		if now.Before(event.Info.Start) && (nextEvent == nil || event.Info.Start.Before(nextEvent.Info.Start)) && data.RuleMatchingEvent(event) != nil {
			nextEvent = event
		}
	}
	return
}
