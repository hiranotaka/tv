package main

import (
	"log"
)

type IdleTask struct {
}

func (task *IdleTask) Run(done <-chan struct{}) error {
	log.Print("Yielding...")
	<-done
	return nil
}
