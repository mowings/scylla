package scheduler

import (
	"github.com/mowings/scylla/scyd/config"
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

func loadJobs(jobs *JobList) (err error) {
	files, _ := filepath.Glob(filepath.Join(runDir(), "*.json"))
	for _, fn := range files {
		job, err := loadJob(fn)
		if err != nil {
			log.Printf("Unable to reload job from %s - %s\n", fn, err.Error())
		} else {
			(*jobs)[job.Name] = job
		}
	}
	return nil
}

func runSchedule(load_chan chan string, status_chan chan StatusRequest) {

	jobs := JobList{}
	err := loadJobs(&jobs)
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
					job.save()
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
			}

		case path := <-load_chan:
			log.Println("Got config load request.")
			cfg, err := config.New(path)
			if err != nil {
				log.Printf("Unable to parse %s : %s\n", path, err.Error)
			} else {
				new_jobs := JobList{}
				for name, job := range cfg.Job {
					if jobs[name] == nil {
						log.Printf("Adding new job: %s\n", name)
						new_job, err := New(job)
						if err != nil {
							log.Printf("Error: Unable to create new job: %s: %s\n", name, err.Error())
						} else {
							new_jobs[name] = new_job
							new_job.save()
						}
					} else {
						log.Printf("Updating job: %s\n", name)
						jobs[name].update(job)
						jobs[name].save()
						new_jobs[name] = jobs[name]
					}
				}
				// Delete old job state files
				for name, _ := range jobs {
					if new_jobs[name] == nil {
						log.Printf("Removing old job file for %s\n", name)
						os.Remove(filepath.Join(runDir(), name+".json"))
					}
				}
				jobs = nil // Go garbage collection in maps can ve weird. Easiest to nil out the old map
				jobs = new_jobs
			}
		}
	}
}