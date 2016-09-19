package main

import (
	"encoding/gob"
	"errors"
	"golang.org/x/exp/inotify"
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
	return nil
}

func listenData(cancel <-chan struct{}, notificationQueue chan<- struct{}) error {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.AddWatch(".data", inotify.IN_MOVED_TO)
	if err != nil {
		return err
	}
	defer watcher.Close()

	notificationQueue <- struct{}{}

	for {
		select {
		case event := <-watcher.Event:
			if event.Name == ".data/tvctl.gob" {
				notificationQueue <- struct{}{}
			}
		case err := <-watcher.Error:
			return err
		case <-cancel:
			return errors.New("Cancelled")
		}
	}
}
