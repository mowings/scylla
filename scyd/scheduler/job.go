package scheduler

import (
	"encoding/json"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/ssh"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

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
	config.JobSpec
	Running         bool
	RunId           int
	RunsOutstanding int
	Runs            []*RunData
	RunsQueued      int
	LastChecked     time.Time
	PoolIndex       int
}

type JobList map[string]*Job

func New(spec *config.JobSpec) (*Job, error) {
	job := Job{}
	err := job.update(spec)
	return &job, err
}

func loadJob(path string) (job *Job, err error) {
	var data []byte
	var new_job Job
	job = &new_job
	if data, err = ioutil.ReadFile(path); err != nil {
		return job, err
	}
	if err := json.Unmarshal(data, job); err != nil {
		return job, err
	}
	// Schedule will be unparsed, so parse it for each job
	err = job.ParseSchedule()
	return job, err
}

func (job *Job) save() (err error) {
	path := filepath.Join(runDir(), job.Name+".json")
	os.MkdirAll(runDir(), 0755)
	var b []byte
	if b, err = json.Marshal(job); err == nil {
		err = ioutil.WriteFile(path, b, 0644)
	}
	return err
}

func (job *Job) saveRuns(runs []*RunData) (err error) {
	run_dir := filepath.Join(runDir(), job.Name, strconv.Itoa(job.RunId))
	os.MkdirAll(run_dir, 0755)
	path := filepath.Join(run_dir, "runs.json")
	var b []byte
	if b, err = json.Marshal(runs); err == nil {
		err = ioutil.WriteFile(path, b, 0644)
	}
	return err
}

func (job *Job) complete(r *RunData) bool {
	job.Runs = append(job.Runs, r)
	if len(job.Runs) == cap(job.Runs) {
		job.Running = false
		job.saveRuns(job.Runs)
		job.RunId += 1
		log.Printf("Completed job %s.%d.\n", job.Name, job.RunId)
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
		job.save()
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

func (job *Job) run(run_report_chan chan *RunData) {
	if job.Running {
		job.RunsQueued += 1
		return
	}
	job.Runs = make([]*RunData, 0, 1)
	job.Running = true
	var host string
	if job.Host != "" {
		host = job.Host
	} else {
		host = job.PoolInst.Host[job.PoolIndex]
	}
	sudo := job.Sudo
	reports := make([]CommandRunData, len(job.Command))
	r := RunData{job.Name, job.RunId, Succeeded, host, reports}
	for index, command := range job.Command {
		reports[index] = CommandRunData{command, "", "", 0, time.Now(), time.Now()}
	}
	keyfile := job.Keyfile
	connection_timeout := job.ConnectTimeout
	run_timeout := job.RunTimeout
	run_dir := filepath.Join(runDir(), job.Name, strconv.Itoa(job.RunId))

	go func() {
		os.MkdirAll(run_dir, 0755)
		log.Printf("Opening connection to: %s (%d)\n", host, connection_timeout)
		conn, err := openConnection(keyfile, host, connection_timeout)
		if err != nil {
			reports[0].Error = err.Error() // Just set first command to error on a failed connection
			log.Printf("Unable to connect to %s (%s)\n", host, err.Error())
		} else {
			defer conn.Close()
			for index, report := range reports {
				command_dir := filepath.Join(run_dir, host, strconv.Itoa(index))
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

func (job *Job) isTimeForJob() bool {
	now := time.Now()
	if time.Since(job.LastChecked) > time.Minute {
		schedule := job.ScheduleInst
		job.LastChecked = now
		if schedule.Match(&now) {
			return true
		}
	}
	return false
}

func (job *Job) update(spec *config.JobSpec) error {
	job.JobSpec = *spec
	var t time.Time
	job.LastChecked = t
	job.PoolIndex = 0
	return nil
}
