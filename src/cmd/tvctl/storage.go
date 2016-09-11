package main

import (
	"encoding/gob"
	"log"
	"net"
	"os"
	"syscall"
	"zng.jp/tv"
)

func readData() (*tv.Data, error) {
	data := &tv.Data{}

	in, err := os.Open(".data/tvctl.gob")
	if err == nil {
		defer in.Close()
		if err := gob.NewDecoder(in).Decode(data); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return data, nil
}

func notifyData() {
	connection, err := net.Dial("unix", ".data/tvctl.sock")
	if err != nil {
		return
	}

	connection.Close()
}

func writeData(newData *tv.Data) error {
	lock, err := syscall.Open(".data/tvctl.lock", syscall.O_WRONLY|syscall.O_CREAT|syscall.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer syscall.Close(lock)

	if err := syscall.Flock(lock, 1); err != nil {
		return err
	}

	data, err := readData()
	if err != nil {
		return err
	}

	data.MergeData(newData)

	out, err := os.Create(".data/tvctl.gob.tmp")
	if err != nil {
		return err
	}

	defer out.Close()
	if err := gob.NewEncoder(out).Encode(data); err != nil {
		return err
	}

	out.Sync()

	if err := os.Rename(".data/tvctl.gob.tmp", ".data/tvctl.gob"); err != nil {
		return err
	}

	notifyData()
	return nil
}

func listenData(cancel <-chan struct{}) (<-chan struct{}, error) {
	os.Remove(".data/tvctl.sock")
	listener, err := net.Listen("unix", ".data/tvctl.sock")
	if err != nil {
		return nil, err
	}

	go func() {
		<-cancel
		listener.Close()
	}()

	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		acceptDone <- struct{}{}
		for {
			connection, err := listener.Accept()
			if err != nil {
				log.Print("Accept failed: %v", err)
				return
			}

			acceptDone <- struct{}{}
			connection.Close()
		}
	}()

	return acceptDone, nil
}
