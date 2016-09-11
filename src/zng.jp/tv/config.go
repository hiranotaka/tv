package tv

var (
	StreamConfigData = &Data{
		StreamConfigMap: map[StreamId]*StreamConfig{
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
		},
	}
)
