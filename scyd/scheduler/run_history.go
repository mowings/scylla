package scheduler

import (
	"time"
)

type RunInfo struct {
	Status    RunStatus
	StartTime time.Time
	EndTime   time.Time
}

type CommandRun struct {
	RunInfo
	CommandSpecified string
	CommandRun       string
	Error            string
	StatusCode       int
	StdOutURI        string `json:",omitempty"`
	StdErrURI        string `json:",omitempty"`
}

type HostRun struct {
	RunInfo
	JobName     string
	RunId       int
	Host        string
	HostId      int
	CommandRuns []CommandRun
}

type JobRun struct {
	RunInfo
	RunId     int
	JobName   string `json:",omitempty"`
	HostRuns  []HostRun
	DetailURI string `json:",omitempty"`
}

type JobHistory []JobRun

// Sortable interface
func (slice JobHistory) Len() int {
	return len(slice)
}

func (slice JobHistory) Less(i, j int) bool {
	return slice[i].RunId < slice[j].RunId
}

func (slice JobHistory) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func (jr JobRun) GetHostRunById(id int) *HostRun {
	for _, hr := range jr.HostRuns {
		if hr.HostId == id {
			return &hr
		}
	}
	return nil
}

func (jr *JobRun) updateStatus() {
	if jr.Status != Running {
		return
	}
	completed := 0
	for _, hr := range jr.HostRuns {
		if hr.Status == Failed {
			jr.Status = Failed
		}
		if hr.Status != Running {
			completed += 1
		}
	}
	if completed == len(jr.HostRuns) {
		jr.Status = Succeeded
		jr.EndTime = time.Now()
	}
}
