package scheduler

import (
	"log"
	"time"
)

const TIMEOUT = 10

func Run() (load_chan chan string, status_chan chan StatusRequest) {
	load_chan = make(chan string)
	status_chan = make(chan StatusRequest)
	go runSchedule(load_chan, status_chan)
	return load_chan, status_chan
}

func runSchedule(load_chan chan string, status_chan chan StatusRequest) {
	for {
		select {
		case <-time.After(time.Second * TIMEOUT):
			log.Println("boom")
		case _ = <-load_chan:
			log.Println("Got config load request.")
		}

	}
}
