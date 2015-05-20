package scheduler

import (
	"errors"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/cronsched"
	"github.com/mowings/scylla/sched"
	"github.com/mowings/scylla/ssh"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

type CommandRunData struct {
	CommandSpecified string
	CommandRun       string
	Error            string
	StatusCode       int
	StartTime        time.Time
	EndTime          time.Time
}

// Message -- run status
type RunData struct {
	JobName     string
	RunId       int
	Status      RunStatus
	Host        string
	CommandRuns []CommandRunData
}

type StatusResponse struct {
	Runs []RunData
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
	Schedule        sched.Sched `json:"-"`
	Running         bool
	RunId           int
	RunsOutstanding int
	Runs            []*RunData
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

func (job *Job) Complete(r *RunData) bool {
	job.Runs = append(job.Runs, r)
	if len(job.Runs) == cap(job.Runs) {
		job.Running = false
		log.Printf("Completed job %s.%d.\n", job.Name, job.RunId)
		job.RunId += 1
		for _, run_report := range job.Runs {
			log.Printf("%s.%d.%s\n", run_report.JobName, run_report.RunId, run_report.Host)
			for _, command_run_report := range run_report.CommandRuns {
				log.Printf("   %s (%s) err = %s\n", command_run_report.CommandSpecified, command_run_report.CommandRun, command_run_report.Error)
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

func (job *Job) Run(run_report_chan chan *RunData) {
	if job.Running {
		job.RunsQueued += 1
		return
	}
	job.Runs = make([]*RunData, 0, 1)
	job.Running = true
	host := qualifyHost(job.Pool[job.PoolIndex], job.Defaults.User, job.Defaults.Port)
	sudo := job.JobSpec.Sudo
	reports := make([]CommandRunData, len(job.JobSpec.Command))
	r := RunData{job.Name, job.RunId, Succeeded, host, reports}
	for index, command := range job.JobSpec.Command {
		reports[index] = CommandRunData{command, "", "", 0, time.Now(), time.Now()}
	}
	keyfile := job.Defaults.Keyfile
	connection_timeout := job.Defaults.ConnectTimeout
	run_timeout := job.JobSpec.RunTimeout
	run_dir := filepath.Join(job.Defaults.RunDir, job.Name, strconv.Itoa(job.RunId))

	go func() {
		os.MkdirAll(run_dir, 0755)
		conn, err := openConnection(keyfile, host, connection_timeout)
		defer conn.Close()
		if err != nil {
			reports[0].Error = err.Error() // Just set first command to error on a failed connection
		} else {
			for index, report := range reports {
				command_dir := filepath.Join(run_dir, strconv.Itoa(index))
				os.MkdirAll(command_dir, 0775)
				reports[index].StartTime = time.Now()
				log.Printf("%s.%d - running command \"%s\" on host %s\n", r.JobName, r.RunId, report.CommandSpecified, host)
				stdout, stderr, err := conn.Run(report.CommandSpecified, run_timeout, sudo)
				if err != nil {
					reports[index].Error = err.Error()
					reports[index].StatusCode = -1
				}
				if stdout != nil {
					ioutil.WriteFile(filepath.Join(command_dir, "stdout"), []byte(*stdout), 0644)
				}
				if stderr != nil {
					ioutil.WriteFile(filepath.Join(command_dir, "stderr"), []byte(*stderr), 0644)
				}
				reports[index].CommandRun = report.CommandSpecified
				reports[index].EndTime = time.Now()
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
	if err := job.ParseSchedule(); err != nil {
		return err
	}
	var t time.Time
	job.LastChecked = t
	err := job.UpdatePool(cfg)
	return err
}

func (job *Job) ParseSchedule() error {
	jobspec := job.JobSpec
	m := rexSched.FindStringSubmatch(jobspec.Schedule)
	if m == nil {
		return errors.New("Unable to parse schedule: " + jobspec.Schedule)
	}
	var schedule sched.Sched
	if m[1] == "cron" {
		schedule = &cronsched.ParsedCronSched{}
	} else {
		return errors.New("Unknown schedule type: " + jobspec.Schedule)
	}
	job.Schedule = schedule
	err := schedule.Parse(m[2])
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