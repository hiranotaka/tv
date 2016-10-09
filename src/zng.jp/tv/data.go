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
	"00003": &StreamConfig{System: ISDB_T, Frequency: 545142857},
	"00004": &StreamConfig{System: ISDB_T, Frequency: 539142857},
	"00005": &StreamConfig{System: ISDB_T, Frequency: 527142857},
	"00006": &StreamConfig{System: ISDB_T, Frequency: 533142857},
	"00007": &StreamConfig{System: ISDB_T, Frequency: 521142857},
	"00008": &StreamConfig{System: ISDB_T, Frequency: 491142857},
	"00009": &StreamConfig{System: ISDB_T, Frequency: 563142857},
	"00010": &StreamConfig{System: ISDB_S, Frequency: 1318000000, TsId: 0x40f1},
	"00011": &StreamConfig{System: ISDB_S, Frequency: 1318000000, TsId: 0x40f2},
	"00012": &StreamConfig{System: ISDB_S, Frequency: 1279640000, TsId: 0x40d0},
	"00013": &StreamConfig{System: ISDB_S, Frequency: 1049480000, TsId: 0x4010},
	"00014": &StreamConfig{System: ISDB_S, Frequency: 1049480000, TsId: 0x4011},
	"00015": &StreamConfig{System: ISDB_S, Frequency: 1087840000, TsId: 0x4031},
	"00016": &StreamConfig{System: ISDB_S, Frequency: 1279640000, TsId: 0x40d1},
	"00017": &StreamConfig{System: ISDB_S, Frequency: 1087840000, TsId: 0x4030},
	"00018": &StreamConfig{System: ISDB_S, Frequency: 1202920000, TsId: 0x4091},
	"00019": &StreamConfig{System: ISDB_S, Frequency: 1202920000, TsId: 0x4090},
	"00020": &StreamConfig{System: ISDB_S, Frequency: 1202920000, TsId: 0x4092},
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

func (data *Data) StreamWithoutStateOrWithOldestState() *Stream {
	for _, stream := range data.Streams() {
		if stream.State == nil {
			return stream
		}
	}

	var streamWithOldestState *Stream
	for _, stream := range data.Streams() {
		if streamWithOldestState == nil || stream.State.Time.Before(streamWithOldestState.State.Time) {
			streamWithOldestState = stream
		}
	}

	return streamWithOldestState
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
