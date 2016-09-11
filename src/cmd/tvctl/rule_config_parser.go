package main

import (
	"net/url"
	"strconv"
	timepkg "time"
	"zng.jp/tv"
)

func parseRuleConfig(values url.Values) (*tv.Data, error) {
	id := values.Get("id")

	deleted := values.Get("deleted") != ""

	var programNumber int64
	programNumberStr := values.Get("program-number")
	if programNumberStr != "" {
		var err error
		programNumber, err = strconv.ParseInt(programNumberStr, 10, 32)
		if err != nil {
			return nil, err
		}
	}

	var start timepkg.Time
	startStr := values.Get("start")
	if startStr != "" {
		var err error
		start, err = timepkg.Parse("2006-01-02 15:04:05.999999999 -0700 MST", startStr)
		if err != nil {
			return nil, err
		}
	}

	return &tv.Data{
		RuleMap: map[tv.RuleId]*tv.Rule{
			tv.RuleId(id): &tv.Rule{
				Config: &tv.RuleConfig{
					Deleted:       deleted,
					ProgramNumber: int32(programNumber),
					Start:         start,
				},
			},
		},
	}, nil
}
