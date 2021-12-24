package main

type Task interface {
	Equals(Task) bool
	Requirements() []int32
	Run(<-chan struct{}, []int32)
}
