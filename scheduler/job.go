package scheduler

import (
	"errors"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/cronsched"
	"github.com/mowings/scylla/sched"
	"log"
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

type CommandRunReport struct {
	CommandSpecified string
	CommandRun       string
	StatusCode       int
	StartTime        time.Time
	EndTime          time.Time
}

// Message -- run status
type RunReport struct {
	JobName           string
	RunId             int
	Status            RunStatus
	Host              string
	CommandRunReports []CommandRunReport
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
	Defaults        *config.Defaults
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
	PoolMode        string
}

type JobList map[string]*Job

func New(cfg *config.Config, name string) (*Job, error) {
	job := Job{}
	job.Name = name
	err := job.Update(cfg)
	return &job, err
}

func (job *Job) Complete(r *RunReport) bool {
	job.RunReports = append(job.RunReports, r)
	if len(job.RunReports) == cap(job.RunReports) {
		job.Running = false
		job.RunId += 1
		log.Printf("Completed job %s.%d.\n", job.Name, job.RunId)
		for _, run_report := range job.RunReports {
			log.Printf("%s.%d.%s\n", run_report.JobName, run_report.RunId, run_report.Host)
			for _, command_run_report := range run_report.CommandRunReports {
				log.Printf("   %s (%s) %d\n", command_run_report.CommandSpecified, command_run_report.CommandRun, command_run_report.StatusCode)
			}
		}
		return true
	}
	return false

}

func (job *Job) Run(run_report_chan chan *RunReport) {
	if job.Running {
		job.RunsQueued += 1
		return
	}
	job.RunReports = make([]*RunReport, 0, 1)
	job.Running = true
	host := qualifyHost(job.Pool[job.PoolCounter], job.Defaults.User, job.Defaults.Port)

	go func() {
		reports := make([]CommandRunReport, len(job.JobSpec.Command))
		r := RunReport{job.Name, job.RunId, Succeeded, host, reports}

		for index, command := range job.JobSpec.Command {
			started := time.Now()
			log.Printf("%s.%d - running command \"%s\" on host %s\n", job.Name, job.RunId, command, host)
			time.Sleep(2 * time.Second)
			reports[index] = CommandRunReport{command, command, 0, started, time.Now()}
		}
		run_report_chan <- &r
	}()

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
	job.Defaults = &cfg.Defaults
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
	job.PoolMode = ""
	if jobspec.Host != "" {
		job.Pool = []string{jobspec.Host}
	} else {
		p := strings.Split(jobspec.Pool, " ")
		job.Pool = cfg.Pool[p[0]].Host
		if len(p) > 1 {
			job.PoolMode = p[1]
		}
	}
	job.PoolCounter = 0

	return nil
}
