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

type JobReport struct {
	Job
	DetailURI string
}

type JobReportWithHistory struct {
	Job
	DetailURI string
	Runs      []RunHistoryReport
}

type JobsByName []JobReport

func (slice JobsByName) Len() int           { return len(slice) }
func (slice JobsByName) Less(i, j int) bool { return slice[i].Name < slice[j].Name }
func (slice JobsByName) Swap(i, j int)      { slice[i], slice[j] = slice[j], slice[i] }

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
	}
	// Sort the jobs before we return them
	sort.Sort(JobsByName(data))
	rchan <- &data
}

func reportJobDetail(jobs *JobList, name string, rchan chan StatusResponse) {
	job := (*jobs)[name]
	if job == nil {
		rchan <- fmt.Sprintf("Job \"%s\" not found.", name)
		return
	}
	j := JobReportWithHistory{Job: *job, Runs: make([]RunHistoryReport, len(job.History))}
	for i, run := range job.History {
		j.Runs[i] = *run.Report(true)
	}
	rchan <- &j
}

func reportJobRunDetail(jobs *JobList, name string, runid string, rchan chan StatusResponse) {
	job := (*jobs)[name]
	if job == nil {
		rchan <- fmt.Sprintf("Job (%s) not found.", name)
		return
	}
	rh := job.getRun(runid)
	if rh == nil {
		rchan <- fmt.Sprintf("Run (%s.%s) not found.", name, runid)
		return
	}
	rchan <- rh.Report(false)
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
		case status_req := <-status_chan:
			switch len(status_req.Object) {
			case 0:
				reportJobList(&jobs, status_req.Chan)
			case 1:
				reportJobDetail(&jobs, status_req.Object[0], status_req.Chan)
			case 2:
				reportJobRunDetail(&jobs, status_req.Object[0], status_req.Object[1], status_req.Chan)
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
