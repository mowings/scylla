package web

import (
	"github.com/mowings/scylla/scyd/scheduler"
)

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
