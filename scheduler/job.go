package scheduler

import (
	"errors"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/cronsched"
	"github.com/mowings/scylla/sched"
	"regexp"
	"strings"
	"time"
)

var rexSched = regexp.MustCompile("^([a-z]+) (.+)")

type RunStatus int

const (
	Succeeded = iota
	Failed
	Waiting
	Abandoned
)

// Message -- run status
type RunReport struct {
	JobName    string
	RunId      int
	Status     RunStatus
	Host       string
	StartTime  time.Time
	EndTime    time.Time
	CommandRun string // Command can be transformed
	StatusCode int
}

type StatusResponse struct {
	RunReports []RunReport
}

type StatusRequest struct {
	Name string
	Chan chan StatusResponse
}

//
type Job struct {
	Name            string
	JobSpec         *config.JobSpec
	Schedule        sched.Sched
	Running         bool
	RunId           int
	RunsOutstanding int
	RunReports      []*RunReport
	RunsQueued      int
	LastChecked     time.Time
	Pool            []string
	PoolCounter     int
}

type JobList map[string]*Job

func New(cfg *config.Config, name string) (*Job, error) {
	job := Job{}
	job.Name = name
	err := job.Update(cfg)
	return &job, err
}

func (job *Job) Complete(r *RunReport) bool {
	return true
}

func (job *Job) Run(run_report_chan chan *RunReport) {

}

func (job *Job) IsTimeForJob() bool {
	now := time.Now()
	if time.Since(job.LastChecked) > time.Minute {
		schedule := job.Schedule
		job.LastChecked = now
		if schedule.Match(&now) {
			return true
		}
	}
	return false
}

func (job *Job) Update(cfg *config.Config) error {
	jobspec := cfg.Job[job.Name]
	job.JobSpec = jobspec
	m := rexSched.FindStringSubmatch(jobspec.Schedule)
	if m == nil {
		errors.New("Unable to parse schedule: " + jobspec.Schedule)
	}
	var schedule sched.Sched
	if m[1] == "cron" {
		schedule = &cronsched.ParsedCronSched{}
	} else {
		return errors.New("Unknown schedule type: " + jobspec.Schedule)
	}
	job.Schedule = schedule
	err := schedule.Parse(m[2])
	var t time.Time
	job.LastChecked = t
	err = job.UpdatePool(cfg)
	return err
}

func (job *Job) UpdatePool(cfg *config.Config) error {
	jobspec := job.JobSpec
	if jobspec.Host != "" {
		job.Pool = []string{jobspec.Host}
	} else {
		p := strings.Split(jobspec.Pool, " ")
		job.Pool = cfg.Pool[p[0]].Host
	}
	job.PoolCounter = 0
	return nil
}
