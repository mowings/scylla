package scheduler

import (
	"encoding/json"
	"github.com/mowings/scylla/config"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

const TIMEOUT = 10

func Run() (load_chan chan string, status_chan chan StatusRequest) {
	load_chan = make(chan string)
	status_chan = make(chan StatusRequest)
	go runSchedule(load_chan, status_chan)
	return load_chan, status_chan
}

func saveRunState(run_dir string, jobs *JobList) (err error) {
	path := filepath.Join(run_dir, "runstate.json")
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var b []byte
	if b, err = json.Marshal(jobs); err != nil {
		return err
	}
	err = ioutil.WriteFile(path, b, 0644)
	return err
}

func runSchedule(load_chan chan string, status_chan chan StatusRequest) {
	jobs := JobList{}
	var cur_config *config.Config
	run_report_chan := make(chan *RunData)
	for {
		select {
		case <-time.After(time.Second * TIMEOUT):
			for name, job := range jobs {
				if job.IsTimeForJob() {
					log.Printf("Time for job: %s\n", name)
					job.Run(run_report_chan)
					saveRunState(cur_config.Defaults.RunDir, &jobs)
				}
			}
		case run_report := <-run_report_chan:
			job := jobs[run_report.JobName]
			if job == nil || job.RunId != run_report.RunId {
				log.Printf("Received run report for unknown job/run id: %s (%d). Discarding\n", run_report.JobName, run_report.RunId)
				break
			}
			if job.Complete(run_report) {
				log.Printf("Completed job %s\n", job.Name)
			}
			saveRunState(cur_config.Defaults.RunDir, &jobs)

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
				saveRunState(cur_config.Defaults.RunDir, &jobs)
			}
		}
	}
}
