package job

import (
	"errors"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/cronsched"
	"github.com/mowings/scylla/sched"
	"regexp"
	"time"
)

var rexSched = regexp.MustCompile("^([a-z) (.+)")

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

//
type Job struct {
	Name            string
	JobSpec         *config.JobSpec
	Schedule        *sched.Sched
	Running         bool
	RunId           int
	RunsOutstanding int
	RunReports      []*RunReport
	RunsQueued      int
	LastChecked     time.Time
	PoolCounter     int
}

func New(jobspec *config.JobSpec, name string) (*Job, error) {
	job := Job{}
	job.Name = name
	err := job.Update(jobspec)
	return &job, err
}

func (job *Job) Update(jobspec *config.JobSpec) error {
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
	job.Schedule = &schedule
	err := schedule.Parse(jobspec.Schedule)
	var t time.Time
	job.LastChecked = t
	return err
}
