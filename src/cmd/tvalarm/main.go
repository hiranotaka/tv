package main

import (
	"log"
	"os/exec"
	"time"
	"zng.jp/tv"
	"zng.jp/tv/db"
)

type state struct {
	alive bool
	end   time.Time
}

func getState(data *tv.Data, now time.Time) *state {
	if event := data.CurrentMatchedEvent(now); event != nil {
		return &state {
			alive: true,
			end: event.End(),
		}
	}

	if event := data.NextMatchedEvent(now); event != nil {
		return &state {
			alive: false,
			end: event.Info.Start,
		}
	}

	return &state {
		alive: false,
		end: now.Add(24 * time.Hour),
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

	var state *state

	for {
		newState := getState(data, time.Now().Add(-5 * time.Minute))
		if (state == nil || !state.alive) && newState.alive {
			log.Printf("Waking up TV...")
			if err := exec.Command("wakeonlan", "d8:cb:8a:e7:bc:ab").Run(); err != nil {
				log.Printf("wakeonlan failed: %v", err)
				break
			}
		}
		state = newState

		timer := time.NewTimer(state.end.Sub(time.Now()))

		log.Printf("Yielding...")

		select {
		case <-timer.C:
			state = nil
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


