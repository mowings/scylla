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

// Response
type StatusResponse interface{}

// Base request
type Request interface{}

// Status Request (report job, run or host/command details)
type StatusRequest struct {
	Object []string
	Chan   chan StatusResponse
}

// Load config request
type LoadConfigRequest string

// Run a job
type RunJobRequest string

// Change Job Status
type ChangeJobStatusRequest struct {
	Name   string
	Status RunStatus
}

type UpdatePoolRequest struct {
	Name  string
	Hosts []string
}

func Run() (request_chan chan Request) {
	request_chan = make(chan Request)
	go runSchedule(request_chan)
	return request_chan
}

func loadJobs(jobs *JobList) (err error) {
	files, _ := filepath.Glob(filepath.Join(config.JobDir(), "*.json"))
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

func runSchedule(request_chan chan Request) {
	var current_config *config.Config
	jobs := JobList{}
	log.Printf("Loading saved job state...")
	err := loadJobs(&jobs)
	if err != nil {
		log.Printf("NOTE: Unable to open jobs state file: %s\n", err.Error())
		jobs = JobList{}
	}
	log.Printf("Done..")

	run_report_chan := make(chan *HostRun)

	for _, job := range jobs {
		if job.RunOnStart {
			log.Printf("Running on-start job: %s", job.Name)
			job.run(run_report_chan)
		}
	}

	for {
		select {
		case <-time.After(time.Second * TIMEOUT): // Check for job runs
			for name, job := range jobs {
				if job.isTimeForJob() {
					log.Printf("Time for job: %s\n", name)
					job.run(run_report_chan)
					job.save()
				}
			}
		case base_req := <-request_chan: // Client requests/commands
			switch req := base_req.(type) {
			case UpdatePoolRequest:
				log.Printf("Received update for pool: %s", req.Name)
				pool := config.PoolSpec{Name: req.Name}
				if current_config != nil {
					pool.UpdateHosts(req.Hosts, current_config.Defaults.User, current_config.Defaults.Port)
				} else {
					pool.Host = req.Hosts // Possible that we have no config data
				}
				for _, job := range jobs {
					if job.PoolInst != nil && job.PoolInst.Name == pool.Name {
						log.Printf("Updating job %s with updated pool %s", job.Name, pool.Name)
						job.PoolInst = &pool
						job.PoolIndex = 0
					}
				}

			case ChangeJobStatusRequest:
				log.Printf("Job status change for: %s", req.Name)
				job := jobs[req.Name]
				if job != nil {
					log.Printf("Changing status of job %s to %d", req.Name, req.Status)
					job.Status = req.Status
					job.save()
				}
			case RunJobRequest:
				name := string(req)
				log.Printf("Manual job run request for: %s", name)
				job := jobs[name]
				if job != nil {
					job.run(run_report_chan)
					job.save()
				}
			case LoadConfigRequest:
				log.Println("Got config load request.")
				path := string(req)
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
							os.Remove(filepath.Join(config.JobDir(), name+".json"))
						}
					}
					jobs = nil // Go garbage collection in maps can ve weird. Easiest to nil out the old map
					jobs = new_jobs

					current_config = cfg
				}
			case StatusRequest:
				switch len(req.Object) {
				case 0:
					reportJobList(&jobs, req.Chan)
				case 1:
					reportJobWithHistory(&jobs, req.Object[0], req.Chan)
				case 2:
					reportJobRun(&jobs, req.Object[0], req.Object[1], req.Chan)
				}

			}
		case run_report := <-run_report_chan: // Job status from goproc command runners
			job := jobs[run_report.JobName]
			if job == nil {
				log.Printf("Received run report for unknown job: %s. Discarding\n", run_report.JobName)
				break
			}
			if job.complete(run_report) {
				log.Printf("Completed job %s\n", job.Name)
			}
		}
	}
}
