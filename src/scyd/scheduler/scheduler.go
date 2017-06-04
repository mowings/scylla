package scheduler

import (
	"encoding/json"
	"fmt"
	"github.com/mowings/scylla/scyd/config"
	"io/ioutil"
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

func saveConfig(cfg *config.Config) (err error) {
	path := filepath.Join(config.RunDir(), "config.json")
	os.MkdirAll(config.RunDir(), 0755)
	var b []byte
	if b, err = json.Marshal(cfg); err == nil {
		err = ioutil.WriteFile(path, b, 0644)
	}
	return err
}

func loadConfig() (*config.Config, error) {
	path := filepath.Join(config.RunDir(), "config.json")
	var cfg config.Config
	var err error
	_, err = os.Stat(path)
	if err != nil {
		return nil, err // Assume stat failure is missing file
	}
	var data []byte

	if data, err = ioutil.ReadFile(path); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
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
	dynamic_pools := make(map[string][]string)
	notifiers := make(map[string]*JobNotifier)
	var cur_config *config.Config
	var err error
	if cur_config, err = loadConfig(); err != nil {
		log.Printf("Unable to load last known good config: %s", err.Error())
	} else {
		log.Printf("Reloaded current good config. %d jobs and %d host pools", len(cur_config.Job), len(cur_config.Pool))
	}
	jobs := JobList{}
	log.Printf("Loading saved job state...")
	err = loadJobs(&jobs)
	if err != nil {
		log.Printf("NOTE: Unable to open jobs state file: %s\n", err.Error())
		jobs = JobList{}
	}
	log.Printf("Done..")

	run_report_chan := make(chan HostRun)

	// Run any run-on-start jobs
	for _, job := range jobs {
		if job.RunOnStart {
			job.run(run_report_chan)
		}
	}

	for {
		select {
		case <-time.After(time.Second * TIMEOUT): // Check for job runs
			for _, job := range jobs {
				if job.isTimeForJob() {
					job.run(run_report_chan)
					job.save()
				}
			}
		case base_req := <-request_chan: // Client requests/commands
			switch req := base_req.(type) {
			case UpdatePoolRequest:
				log.Printf("Received update for pool: %s", req.Name)
				hosts := req.Hosts
				dynamic_pools[req.Name] = hosts
				for _, job := range jobs {
					if job.PoolInst != nil && job.PoolInst.Name == req.Name && job.PoolInst.Dynamic {
						log.Printf("Updating job %s with updated pool %s", job.Name, req.Name)
						job.PoolInst.Host = hosts
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
					// If the config has dymamic pools, update them from any current dynamic pools
					for name, pool := range cfg.Pool {
						if pool.Dynamic && dynamic_pools[name] != nil {
							cfg.Pool[name].Host = dynamic_pools[name]
						}
					}
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
					notifiers = make(map[string]*JobNotifier)
					for name, notifier := range cfg.Notifier {
						notifiers[name] = &JobNotifier{*notifier}
					}
					jobs = nil // Go garbage collection in maps can ve weird. Easiest to nil out the old map
					jobs = new_jobs
					saveConfig(cfg)
					cur_config = cfg
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
			job.complete(&run_report, notifiers[job.Notifier])
		}
	}
}
