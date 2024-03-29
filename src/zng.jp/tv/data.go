package tv

import (
	"errors"
	"fmt"
	"sort"
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
	Duration      time.Duration
	Name          string
	Weekly        bool
}

type StreamState struct {
	Time time.Time
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

type Data struct {
	RuleConfigMap  map[RuleId]*RuleConfig
	StreamStateMap map[StreamId]*StreamState
	StreamInfoMap  map[StreamId]*StreamInfo
}

type Stream struct {
	Id     StreamId
	Config *StreamConfig
	State  *StreamState
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

func (event *Event) Id() EventId {
	return EventId(fmt.Sprintf("%05d@%s", event.IndexInProgram, event.Program.Id()))
}

func (event *Event) End() time.Time {
	return event.Info.Start.Add(event.Info.Duration)
}

func (event *Event) IsCurrent(now time.Time) bool {
	return event.Info.Start.Before(now) && now.Before(event.End())
}

func (event *Event) Overlaps(otherEvent *Event) bool {
	return otherEvent.Info.Start.Before(event.End()) && event.Info.Start.Before(otherEvent.End())
}

func (program *Program) Id() ProgramId {
	return ProgramId(fmt.Sprintf("%05d@%s@%s", program.IndexInStream, program.Stream.Info.Time.String(), program.Stream.Id))
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

func (stream *Stream) Url(assignment int32) (string, error) {
	config := stream.Config
	switch config.System {
	case ISDB_T:
		return fmt.Sprintf("isdb-t://adapter=%d:frequency=%d", assignment*2+1, config.Frequency), nil
	case ISDB_S:
		return fmt.Sprintf("isdb-s://adapter=%d:frequency=%d:ts-id=%d", assignment*2, config.Frequency, config.TsId), nil
	default:
		return "", errors.New("Unknown system")
	}
}

func (rule *Rule) MatchEvent(event *Event) bool {
	if event.Program.Info.Number != rule.Config.ProgramNumber {
		return false
	}

	if !rule.Config.Weekly {
		if event.Info.Start != rule.Config.Start {
			return false
		}
	} else {
		if event.Info.Start.Location() != rule.Config.Start.Location() || event.Info.Start.Weekday() != rule.Config.Start.Weekday() || event.Info.Start.Hour() != rule.Config.Start.Hour() || event.Info.Start.Minute() != rule.Config.Start.Minute() {
			return false
		}
	}
	return true
}

func (data *Data) InsertRuleConfig(id RuleId, config *RuleConfig) {
	if data.RuleConfigMap == nil {
		data.RuleConfigMap = make(map[RuleId]*RuleConfig)
	}
	data.RuleConfigMap[id] = config
}

func (data *Data) InsertStreamState(id StreamId, info *StreamState) {
	if data.StreamStateMap == nil {
		data.StreamStateMap = make(map[StreamId]*StreamState)
	}
	data.StreamStateMap[id] = info
}

func (data *Data) InsertStreamInfo(id StreamId, info *StreamInfo) {
	if data.StreamInfoMap == nil {
		data.StreamInfoMap = make(map[StreamId]*StreamInfo)
	}
	data.StreamInfoMap[id] = info
}

func (data *Data) MergeData(newData *Data) {
	for id, newConfig := range newData.RuleConfigMap {
		if !newConfig.Deleted {
			data.InsertRuleConfig(id, newConfig)
		} else {
			delete(data.RuleConfigMap, id)
		}
	}

	for id, newState := range newData.StreamStateMap {
		data.InsertStreamState(id, newState)
	}

	for id, newInfo := range newData.StreamInfoMap {
		data.InsertStreamInfo(id, newInfo)
	}
}

var streamConfigMap = map[StreamId]*StreamConfig{
	"00001": &StreamConfig{System: ISDB_T, Frequency: 557142857},
	"00002": &StreamConfig{System: ISDB_T, Frequency: 551142857},
	"00004": &StreamConfig{System: ISDB_T, Frequency: 545142857},
	"00005": &StreamConfig{System: ISDB_T, Frequency: 539142857},
	"00006": &StreamConfig{System: ISDB_T, Frequency: 527142857},
	"00007": &StreamConfig{System: ISDB_T, Frequency: 533142857},
	"00008": &StreamConfig{System: ISDB_T, Frequency: 521142857},
	"00009": &StreamConfig{System: ISDB_T, Frequency: 491142857},
	"00101": &StreamConfig{System: ISDB_S, Frequency: 1318000000, TsId: 0x40f1},
	"00103": &StreamConfig{System: ISDB_S, Frequency: 1087840000, TsId: 0x4031},
	"00141": &StreamConfig{System: ISDB_S, Frequency: 1279640000, TsId: 0x40d0},
	"00151": &StreamConfig{System: ISDB_S, Frequency: 1049480000, TsId: 0x4010},
	"00161": &StreamConfig{System: ISDB_S, Frequency: 1049480000, TsId: 0x4011},
	"00171": &StreamConfig{System: ISDB_S, Frequency: 1049480000, TsId: 0x4012},
	"00181": &StreamConfig{System: ISDB_S, Frequency: 1279640000, TsId: 0x40d1},
	"00191": &StreamConfig{System: ISDB_S, Frequency: 1087840000, TsId: 0x4030},
	"00192": &StreamConfig{System: ISDB_S, Frequency: 1126200000, TsId: 0x4450},
	"00193": &StreamConfig{System: ISDB_S, Frequency: 1126200000, TsId: 0x4451},
	"00200": &StreamConfig{System: ISDB_S, Frequency: 1202920000, TsId: 0x4091},
	"00201": &StreamConfig{System: ISDB_S, Frequency: 1318000000, TsId: 0x40f2},
	"00211": &StreamConfig{System: ISDB_S, Frequency: 1202920000, TsId: 0x4090},
	"00222": &StreamConfig{System: ISDB_S, Frequency: 1202920000, TsId: 0x4092},
	"00231": &StreamConfig{System: ISDB_S, Frequency: 1241280000, TsId: 0x46b2},
	"00234": &StreamConfig{System: ISDB_S, Frequency: 1394720000, TsId: 0x4730},
	"00236": &StreamConfig{System: ISDB_S, Frequency: 1279640000, TsId: 0x46d2},
	"00241": &StreamConfig{System: ISDB_S, Frequency: 1241280000, TsId: 0x46b1},
	"00242": &StreamConfig{System: ISDB_S, Frequency: 1394720000, TsId: 0x4731},
	"00243": &StreamConfig{System: ISDB_S, Frequency: 1394720000, TsId: 0x4732},
	"00244": &StreamConfig{System: ISDB_S, Frequency: 1433080000, TsId: 0x4751},
	"00245": &StreamConfig{System: ISDB_S, Frequency: 1433080000, TsId: 0x4752},
	"00251": &StreamConfig{System: ISDB_S, Frequency: 1471330000, TsId: 0x4770},
	"00252": &StreamConfig{System: ISDB_S, Frequency: 1433080000, TsId: 0x4750},
	"00255": &StreamConfig{System: ISDB_S, Frequency: 1471330000, TsId: 0x4771},
	"00256": &StreamConfig{System: ISDB_S, Frequency: 1087840000, TsId: 0x4632},
}

func (data *Data) Streams() (streams []*Stream) {
	for id, config := range streamConfigMap {
		streams = append(streams, &Stream{
			Id:     id,
			Config: config,
			State:  data.StreamStateMap[id],
			Info:   data.StreamInfoMap[id],
		})
	}
	return
}

func (data *Data) Programs() (programs []*Program) {
	for _, stream := range data.Streams() {
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

func (data *Data) Rules() (rules []*Rule) {
	for id, config := range data.RuleConfigMap {
		rules = append(rules, &Rule{
			Id:     id,
			Config: config,
		})
	}
	return
}

func (data *Data) FindEvent(id EventId) *Event {
	for _, event := range data.Events() {
		if event.Id() == id {
			return event
		}
	}
	return nil
}

func (data *Data) RuleMatchingEvent(event *Event) *Rule {
	for _, rule := range data.Rules() {
		if rule.Config == nil {
			continue
		}
		if rule.MatchEvent(event) {
			return rule
		}
	}
	return nil
}

type operation struct {
	t            time.Time
	addedEvent   *Event
	removedEvent *Event
}
type operationsByTime []*operation

func (operations operationsByTime) Len() int {
	return len(operations)
}
func (operations operationsByTime) Swap(i, j int) {
	operations[i], operations[j] = operations[j], operations[i]
}
func (operations operationsByTime) Less(i, j int) bool {
	return operations[i].t.Before(operations[j].t)
}

func (data *Data) OverlappingMatchedEvents(theEvent *Event) []*Event {
	operations := []*operation{}
	for _, event := range data.Events() {
		if event.Info == nil {
			continue
		}
		if event.Program.Stream.Config.System != theEvent.Program.Stream.Config.System {
			continue
		}
		if !theEvent.Overlaps(event) {
			continue
		}
		if data.RuleMatchingEvent(event) == nil {
			continue
		}
		operations = append(operations, &operation{
			t:          event.Info.Start,
			addedEvent: event,
		}, &operation{
			t:            event.End(),
			removedEvent: event,
		})
	}

	sort.Sort(operationsByTime(operations))

	resources := 2
	events := []*Event{}
	for _, operation := range operations {
		if operation.addedEvent != nil {
			events = append(events, operation.addedEvent)
			resources--
			if resources <= 0 {
				return events
			}
		} else if operation.removedEvent != nil {
			resources++
			for i, event := range events {
				if event == operation.removedEvent {
					events[i] = events[len(events)-1]
					events = events[0 : len(events)-1]
				}
			}
		}
	}
	return nil
}
