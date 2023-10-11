package main

import "time"

type Timer struct {
	startTime float64
}

func makeTimer() Timer {
	timer := Timer{}
	timer.start()
	return timer
}

//lint:ignore U1000 for debugging
func (t *Timer) start() {
	t.startTime = curTime()
}

//lint:ignore U1000 for debugging
func (t *Timer) tick() float64 {
	curTime := curTime()
	elapsed := curTime - t.startTime
	t.startTime = curTime
	return elapsed
}

//lint:ignore U1000 for debugging
func (t *Timer) elapsed() float64 {
	return curTime() - t.startTime
}

func curTime() float64 {
	return float64(time.Now().UnixNano()) / 1e9
}
