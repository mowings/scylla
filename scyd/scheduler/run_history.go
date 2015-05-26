package scheduler

import (
	"time"
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

type RunHistory struct {
	RunId int
	Runs  []*RunData
}

type CommandRunReport struct {
	CommandRunData
	StdOutURI string
	StdErrURI string
}

type HostRunReport struct {
	Status      RunStatus
	Host        string
	CommandRuns []CommandRunReport
}

type RunHistoryReport struct {
	RunId     int
	JobName   string `json:",omitempty"`
	HostRuns  []HostRunReport
	DetailURI string
}

type JobHistory []RunHistory

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

func (rh *RunHistory) Report(omitjobname bool) RunHistoryReport {
	report := RunHistoryReport{RunId: rh.RunId, HostRuns: make([]HostRunReport, len(rh.Runs))}
	if !omitjobname {
		report.JobName = rh.Runs[0].JobName
	}
	for i, run := range rh.Runs {
		report.HostRuns[i].Status = run.Status
		report.HostRuns[i].Host = run.Host
		report.HostRuns[i].CommandRuns = make([]CommandRunReport, len(run.CommandRuns))
		for j, command_run := range run.CommandRuns {
			report.HostRuns[i].CommandRuns[j] = CommandRunReport{CommandRunData: command_run}
		}
	}
	return report
}
