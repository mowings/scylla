package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/ssh"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type RunStatus int

const (
	None RunStatus = iota
	Running
	Succeeded
	Failed
	Cancelled
	Abandoned
)

type JobReport struct {
	Job
	DetailURI string
}

type JobReportWithHistory struct {
	Job
	PoolHosts []string
	DetailURI string
	Runs      JobHistory
}

type StatusResponse interface{}

type StatusRequest struct {
	Object []string
	Chan   chan StatusResponse
}

// Job runtime
type Job struct {
	config.JobSpec
	RunInfo
	RunId           int
	RunsOutstanding int
	RunsQueued      int
	LastChecked     time.Time
	PoolIndex       int
	History         JobHistory `json:"-"`
}

type JobList map[string]*Job

type JobsByName []JobReport

func (slice JobsByName) Len() int           { return len(slice) }
func (slice JobsByName) Less(i, j int) bool { return slice[i].Name < slice[j].Name }
func (slice JobsByName) Swap(i, j int)      { slice[i], slice[j] = slice[j], slice[i] }

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
	if err != nil {
		return job, err
	}
	lower_bound := job.RunId - job.MaxRunHistory
	job.History = make(JobHistory, 0, 10)
	// Load up job history. Ignore entries earlier than Runid - MaxHistory
	run_path := filepath.Join(runDir(), job.Name, "*")
	run_dirs, _ := filepath.Glob(run_path)
	for _, rd := range run_dirs {
		_, subdir := filepath.Split(rd)
		id, cvt_err := strconv.Atoi(subdir)
		if cvt_err == nil && id >= lower_bound {
			run := JobRun{}
			data, err2 := ioutil.ReadFile(filepath.Join(rd, "run.json"))
			if err2 == nil {
				if err := json.Unmarshal(data, &run); err == nil {
					job.History = append(job.History, run)
				} else {
					log.Printf("Unable to marshal run: %s\n", err.Error())
				}
			} else {
				log.Printf("Unable to read run file: %s\b", err2.Error())
			}
		}
	}
	sort.Sort(sort.Reverse(job.History))
	return job, err
}

func (job *Job) update(spec *config.JobSpec) error {
	job.JobSpec = *spec
	var t time.Time
	job.LastChecked = t
	job.PoolIndex = 0
	if job.Status == Running {
		job.Status = Abandoned
	}
	return nil
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

func (job *Job) saveRun(run *JobRun) (err error) {
	run_dir := filepath.Join(runDir(), job.Name, strconv.Itoa(job.RunId))
	os.MkdirAll(run_dir, 0755)
	path := filepath.Join(run_dir, "run.json")
	var b []byte
	if b, err = json.Marshal(run); err == nil {
		err = ioutil.WriteFile(path, b, 0644)
	}
	return err
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

func cleanHistory(jobname string, runid int) {
	run_dir := filepath.Join(runDir(), jobname, strconv.Itoa(runid))
	log.Printf("Cleaning up directory: %s\n", run_dir)
	go func() {
		os.RemoveAll(run_dir)
	}()
}

func (job *Job) complete(r *HostRun) bool {
	log.Printf("Received host run report %s.%d.%s status=%d\n", job.Name, r.RunId, r.Host, r.Status)
	i, err := job.getRunIndex(r.RunId)
	if err != nil {
		log.Printf("ERROR: %s in job complete", err.Error())
		return false
	}
	for j, hr := range job.History[i].HostRuns {
		if hr.HostId == r.HostId {
			job.History[i].HostRuns[j] = *r
		}
	}
	job.History[i].updateStatus()
	if job.History[0].Status != Running {
		job.Status = job.History[0].Status
		log.Printf("Completed job %s.%d (%d)\n", job.Name, job.RunId, job.Status)
		job.EndTime = time.Now()
		job.RunId += 1
		job.save()
		return true
	}
	return false
}

func (job *Job) getRunIndex(id int) (int, error) {
	for idx, rh := range job.History {
		if rh.RunId == id {
			return idx, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("run %d nost found", id))
}

func (job *Job) getRun(id string) *JobRun {
	nid, err := strconv.Atoi(id)
	if err != nil {
		return nil
	}
	for _, rh := range job.History {
		if rh.RunId == nid {
			return &rh
		}
	}
	return nil
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

func (job *Job) hostRuns() []HostRun {
	var runs []HostRun
	if job.Host != "" {
		runs = make([]HostRun, 1)
		runs[0] = HostRun{JobName: job.Name, RunId: job.RunId, Host: job.Host, HostId: 0}
	} else if job.PoolMode != "parallel" {
		runs = make([]HostRun, 1)
		if job.PoolIndex >= len(job.PoolInst.Host) {
			job.PoolIndex = 0
		}
		runs[0] = HostRun{JobName: job.Name, RunId: job.RunId, Host: job.PoolInst.Host[job.PoolIndex], HostId: 0}
		job.PoolIndex += 1
	} else {
		runs = make([]HostRun, len(job.PoolInst.Host))
		for i, h := range job.PoolInst.Host {
			runs[i] = HostRun{JobName: job.Name, RunId: job.RunId, Host: h, HostId: i}
		}
	}
	for i, _ := range runs {
		runs[i].CommandRuns = make([]CommandRun, len(job.Command))
		for j, cmd := range job.Command {
			runs[i].CommandRuns[j] = CommandRun{CommandSpecified: cmd}
		}
	}
	return runs
}

func (job *Job) run(run_report_chan chan *HostRun) {
	if job.Status == Running {
		job.RunsQueued += 1
		return
	}
	job.StartTime = time.Now()
	job.Status = Running
	runs := job.hostRuns() // Create array of host run objects
	job_run := JobRun{RunId: job.RunId, JobName: job.Name, HostRuns: runs}
	job_run.Status = Running
	job.History = append([]JobRun{job_run}, job.History...)
	l := len(job.History)
	if l > job.MaxRunHistory {
		var old_run JobRun
		old_run, job.History = job.History[l-1], job.History[:l-1]
		cleanHistory(job.Name, old_run.RunId)
	}
	sudo := job.Sudo
	keyfile := job.Keyfile
	connection_timeout := job.ConnectTimeout
	listen_timeout := job.RunTimeout
	run_dir := filepath.Join(runDir(), job.Name, strconv.Itoa(job.RunId))
	for _, run := range runs {
		run.Status = Running
		go runCommandsOnHost(run, sudo, keyfile, connection_timeout, listen_timeout, run_dir, run_report_chan)
	}
}

// Run command set on single remote host
func runCommandsOnHost(
	hr HostRun,
	sudo bool,
	keyfile string,
	connection_timeout int,
	read_timeout int,
	run_dir string,
	run_report_chan chan *HostRun) {
	log.Printf("Opening connection to: %s (%d)\n", hr.Host, connection_timeout)
	hr.StartTime = time.Now()
	hr.Status = Running
	conn, err := openConnection(keyfile, hr.Host, connection_timeout)
	if err != nil {
		hr.CommandRuns[0].Error = err.Error() // Just set first command to error on a failed connection
		hr.CommandRuns[0].Status = Failed
		hr.Status = Failed
		log.Printf("Unable to connect to %s (%s)\n", hr.Host, err.Error())
		hr.EndTime = time.Now()
	} else {
		defer conn.Close()
		for index, report := range hr.CommandRuns {
			command_dir := filepath.Join(run_dir, strconv.Itoa(hr.HostId), strconv.Itoa(index))
			os.MkdirAll(command_dir, 0775)
			hr.CommandRuns[index].StartTime = time.Now()
			log.Printf("%s.%d - running command \"%s\" on host %s\n", hr.JobName, hr.RunId, report.CommandSpecified, hr.Host)
			stdout, stderr, err := conn.Run(report.CommandSpecified, read_timeout, sudo)
			if err != nil {
				hr.CommandRuns[index].Error = err.Error()
				hr.CommandRuns[index].StatusCode = -1
				hr.CommandRuns[index].Status = Failed
				hr.Status = Failed
			} else {
				hr.CommandRuns[index].Status = Succeeded
				if hr.Status == Running {
					hr.Status = Succeeded
				}
			}
			if stdout != nil {
				ioutil.WriteFile(filepath.Join(command_dir, "stdout"), []byte(*stdout), 0644)
			}
			if stderr != nil {
				ioutil.WriteFile(filepath.Join(command_dir, "stderr"), []byte(*stderr), 0644)
			}
			hr.CommandRuns[index].EndTime = time.Now()
		}
		hr.EndTime = time.Now()
	}
	run_report_chan <- &hr
}
