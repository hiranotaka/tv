package main

type Task interface {
	Run(<-chan struct{}) error
}
