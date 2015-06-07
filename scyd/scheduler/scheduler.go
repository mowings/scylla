package scheduler

import (
	"fmt"
	"github.com/mowings/scylla/scyd/config"
	"log"
	"os"
	"path/filepath"
	"sort"
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

func reportJobList(jobs *JobList, rchan chan StatusResponse) {
	data := make([]JobReport, len(*jobs))
	idx := 0
	for _, job := range *jobs {
		data[idx].Job = *job // Make a copy
		idx += 1
	}
	// Sort the jobs before we return them
	sort.Sort(JobsByName(data))
	rchan <- &data
}

func reportJobWithHistory(jobs *JobList, name string, rchan chan StatusResponse) {
	job := (*jobs)[name]
	if job == nil {
		rchan <- fmt.Sprintf("Job \"%s\" not found.", name)
		return
	}
	j := JobReportWithHistory{Job: *job, Runs: make([]JobRun, len(job.History))}
	if job.PoolInst != nil {
		j.PoolHosts = job.PoolInst.Host
	}
	for i, run := range job.History {
		j.Runs[i] = run
	}
	rchan <- &j
}

func reportJobRun(jobs *JobList, name string, runid string, rchan chan StatusResponse) {
	job := (*jobs)[name]
	if job == nil {
		rchan <- fmt.Sprintf("Job (%s) not found.", name)
		return
	}
	jr := job.getRun(runid)
	if jr == nil {
		rchan <- fmt.Sprintf("Run (%s.%s) not found.", name, runid)
		return
	}
	rchan <- jr
}

func runSchedule(load_chan chan string, status_chan chan StatusRequest) {
	jobs := JobList{}
	log.Printf("Loading saved job state...")
	err := loadJobs(&jobs)
	if err != nil {
		log.Printf("NOTE: Unable to open jobs state file: %s\n", err.Error())
		jobs = JobList{}
	}
	log.Printf("Done..")

	run_report_chan := make(chan *HostRun)

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
		case status_req := <-status_chan:
			switch len(status_req.Object) {
			case 0:
				reportJobList(&jobs, status_req.Chan)
			case 1:
				reportJobWithHistory(&jobs, status_req.Object[0], status_req.Chan)
			case 2:
				reportJobRun(&jobs, status_req.Object[0], status_req.Object[1], status_req.Chan)
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
