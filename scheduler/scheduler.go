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
	run_report_chan := make(chan *RunReport)
	for {
		select {
		case <-time.After(time.Second * TIMEOUT):
			for name, job := range jobs {
				if job.IsTimeForJob() {
					log.Printf("Time for job: %s\n", name)
					job.Run(run_report_chan)
				}
			}
		case run_report := <-run_report_chan:
			job := jobs[run_report.JobName]
			if job == nil || job.RunId != run_report.RunId {
				log.Printf("Received run report for unknown job/run id: %s (%d). Discarding\n", run_report.JobName, run_report.RunId)
				break
			}
			log.Printf("Command complete for job %s.%d\n", run_report.JobName, run_report.RunId)
			log.Printf("Command : \"%s\" on host %s", run_report.CommandRun, run_report.Host)
			if job.Complete(run_report) {
				log.Printf("Completed job %s\n", job.Name)
			}

		case path := <-load_chan:
			log.Println("Got config load request.")
			cfg, err := config.New(path)
			if err != nil {
				log.Printf("Unable to parse %s : %s\n", path, err.Error)
			} else {
				cur_config = cfg
				for name, _ := range cur_config.Job {
					if jobs[name] == nil {
						log.Printf("Adding new job: %s\n", name)
						new_job, err := New(cfg, name)
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
