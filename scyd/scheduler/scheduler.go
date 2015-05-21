package scheduler

import (
	"encoding/json"
	"github.com/mowings/scylla/scyd/config"
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

func runDir() string {
	path := os.Getenv("SCYLLA_PATH")
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "./"
	}
	if path == "" {
		path = filepath.Join(cwd, "run")
	} else {
		path = filepath.Join(path, "run")
	}
	return path
}

func saveJobState(jobs *JobList) (err error) {
	path := filepath.Join(runDir(), "jobs.json")
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

func loadRunState(jobs *JobList) (err error) {
	path := filepath.Join(runDir(), "runstate.json")
	var data []byte
	if data, err = ioutil.ReadFile(path); err != nil {
		return err
	}
	if err := json.Unmarshal(data, jobs); err != nil {
		return err
	}
	// Schedule will be unparsed, so parse it for each job
	for _, job := range *jobs {
		job.parseSchedule()
	}
	return nil
}

func runSchedule(load_chan chan string, status_chan chan StatusRequest) {

	jobs := JobList{}
	err := loadRunState(&jobs)
	if err != nil {
		log.Printf("NOTE: Unable to open jobs state file: %s\n", err.Error())
		jobs = JobList{}
	}
	run_report_chan := make(chan *RunData)

	for {
		select {
		case <-time.After(time.Second * TIMEOUT):
			for name, job := range jobs {
				if job.isTimeForJob() {
					log.Printf("Time for job: %s\n", name)
					job.run(run_report_chan)
					saveJobState(&jobs)
				}
			}
		case run_report := <-run_report_chan:
			job := jobs[run_report.JobName]
			if job == nil || job.RunId != run_report.RunId {
				log.Printf("Received run report for unknown job/run id: %s (%d). Discarding\n", run_report.JobName, run_report.RunId)
				break
			}
			if job.complete(run_report) {
				log.Printf("Completed job %s\n", job.Name)
				if err = saveJobState(&jobs); err != nil {
					log.Printf("ERROR: Unable to save job status: %s\n", err.Error())
				}
			}

		case path := <-load_chan:
			log.Println("Got config load request.")
			cfg, err := config.New(path)
			if err != nil {
				log.Printf("Unable to parse %s : %s\n", path, err.Error)
			} else {
				new_jobs := JobList{}
				for name, _ := range cfg.Job {
					if jobs[name] == nil {
						log.Printf("Adding new job: %s\n", name)
						new_job, err := New(cfg, name)
						if err != nil {
							log.Printf("Error: Unable to create new job: %s: %s\n", name, err.Error())
						} else {
							new_jobs[name] = new_job
						}
					} else {
						log.Printf("Updating job: %s\n", name)
						jobs[name].update(cfg)
						new_jobs[name] = jobs[name]
					}
				}
				jobs = nil // Go garbage collection in maps can ve weird. Easiest to nil out the old map
				jobs = new_jobs
				saveJobState(&jobs)
			}
		}
	}
}
