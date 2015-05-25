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
