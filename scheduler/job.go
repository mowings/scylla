package scheduler

import (
	"errors"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/cronsched"
	"github.com/mowings/scylla/sched"
	"github.com/mowings/scylla/ssh"
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
	Error            string
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
	PoolIndex       int
	PoolMode        string
	ConnectionCache map[string]*ssh.SshConnection
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
				log.Printf("   %s (%s) %s\n", command_run_report.CommandSpecified, command_run_report.CommandRun, command_run_report.Error)
			}
		}
		job.PoolIndex += 1
		if job.PoolIndex >= len(job.Pool) {
			job.PoolIndex = 0
		}
		return true
	}
	return false

}

func openConnection(keyfile string, host string, timeout int) (*ssh.SshConnection, error) {
	auths := ssh.MakeKeyring([]string{keyfile})
	var c ssh.SshConnection
	err := c.Open(host, auths, timeout)
	if err != nil {
		return nil, err
	}
	return &c, err
}

func (job *Job) Run(run_report_chan chan *RunReport) {
	if job.Running {
		job.RunsQueued += 1
		return
	}
	job.RunReports = make([]*RunReport, 0, 1)
	job.Running = true
	host := qualifyHost(job.Pool[job.PoolIndex], job.Defaults.User, job.Defaults.Port)

	reports := make([]CommandRunReport, len(job.JobSpec.Command))
	r := RunReport{job.Name, job.RunId, Succeeded, host, reports}
	for index, command := range job.JobSpec.Command {
		reports[index] = CommandRunReport{command, "", "", 0, time.Now(), time.Now()}
	}
	keyfile := job.Defaults.Keyfile
	connection_timeout := job.Defaults.ConnectTimeout

	go func() {
		conn, err := openConnection(keyfile, host, connection_timeout)
		defer conn.Close()
		if err != nil {
			reports[0].Error = err.Error() // Just set first command to error on a failed connection
		} else {
			for _, report := range reports {
				report.StartTime = time.Now()
				log.Printf("%s.%d - running command \"%s\" on host %s\n", r.JobName, r.RunId, report.CommandSpecified, host)
				stdout, stderr, err := conn.Run(report.CommandSpecified, 0, false)
				if err != nil {
					report.Error = err.Error()
					report.StatusCode = -1
				}
				if stdout != nil {
					log.Println("Stdout\n" + *stdout)
				}
				if stderr != nil {
					log.Println("Sterr\n" + *stderr)
				}
				report.EndTime = time.Now()
			}
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
	job.PoolIndex = 0

	return nil
}
