package web

import (
	"fmt"
	"github.com/mowings/scylla/scyd/scheduler"
	"time"
)

var nilTime = time.Time{}

var status_names = []string{"Succeeded", "Failed", "Abandoned"}

type Helpers struct {
}

func (h Helpers) DisplayBool(val bool) string {
	if val {
		return "yes"
	}
	return "no"
}

func (h Helpers) DisplayRunStatus(status scheduler.RunStatus) string {
	if status < scheduler.Succeeded || status > scheduler.Abandoned {
		return "unknown"
	}
	return status_names[status]
}

func (h Helpers) DisplayAgo(from time.Time) string {
	if from == nilTime {
		return "never"
	}
	return h.DisplayDuration(from, time.Now())
}

func (h Helpers) DisplayDuration(from time.Time, to time.Time) string {
	duration := to.Sub(from)
	if int(duration.Minutes()) <= 0 {
		return "< 1m"
	}
	tm := ""
	days := int(duration.Hours()) / 24
	if days > 0 {
		tm += fmt.Sprintf("%dd", days)
	}
	hours := int(duration.Hours()) % 24
	if hours > 0 {
		tm += fmt.Sprintf("%dh", hours)
	}
	minutes := int(duration.Minutes()) % 60
	tm += fmt.Sprintf("%dm", minutes)
	return tm
}
