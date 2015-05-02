package scheduler

import (
	"github.com/mowings/scylla/config"
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
	jobs := JobList{}
	var cur_config *config.Config
	for {
		select {
		case <-time.After(time.Second * TIMEOUT):
			for name, job := range jobs {
				if time.Since(job.LastChecked) > time.Minute {
					schedule := job.Schedule
					now := time.Now()
					job.LastChecked = now
					if schedule.Match(&now) {
						log.Printf("Time for job: %s\n", name)
					}
				}
			}
		case path := <-load_chan:
			log.Println("Got config load request.")
			cfg, err := config.New(path)
			if err != nil {
				log.Printf("Unable to parse %s : %s\n", path, err.Error)
			} else {
				cur_config = cfg
				for name, job_spec := range cur_config.Job {
					if jobs[name] == nil {
						log.Printf("Adding new job: %s\n", name)
						new_job, err := New(job_spec, name)
						if err != nil {
							log.Printf("Error: Unable to create new job: %s: %s\n", name, err.Error())
						} else {
							jobs[name] = new_job
						}
					} else {
						log.Printf("Updating job: %s\n", name)
					}
				}
			}
		}
	}
}
