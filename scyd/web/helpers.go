package web

import (
	"fmt"
	"github.com/mowings/scylla/scyd/scheduler"
	"html/template"
	"time"
)

var nilTime = time.Time{}

var status_names = []string{"None", "Succeeded", "Failed", "Abandoned"}
var status_class_names = []string{"bg-info", "bg-success", "bg-danger", "bg-warning"}
var status_buttons = []string{
	"<button class=\"btn btn-status btn-small btn-info\">none</button>",
	"<button class=\"btn btn-status btn-small btn-success\">success</button>",
	"<button class=\"btn btn-status btn-small btn-danger\">failed</button>",
	"<button class=\"btn btn-status btn-small btn-warning\">dropped</button>",
}

const BTN_UNKNOWN = "<button class==\"btn btn-status btn-small btn-warning\">unknown</button>"

type Helpers struct {
}

func (h Helpers) DisplayBool(val bool) string {
	if val {
		return "yes"
	}
	return "no"
}

func (h Helpers) DisplayFullRunStatus(status scheduler.RunStatus, running bool) string {
	ret := h.DisplayRunStatus(status)
	if running {
		ret += " (Running)"
	} else {
		ret += " (Idle)"
	}
	return ret
}

func (h Helpers) DisplayRunningHtml(running bool) template.HTML {
	if running {
		return template.HTML("<div class=\"running\">yes</div>")
	}
	return template.HTML("<div class=\"not-running\">no</div>")

}

func (h Helpers) DisplayRunStatus(status scheduler.RunStatus) string {
	if status < scheduler.None || status > scheduler.Abandoned {
		return "unknown"
	}
	return status_names[status]
}

func (h Helpers) DisplayRunStatusClasses(status scheduler.RunStatus) string {
	if status < scheduler.None || status > scheduler.Abandoned {
		return "bg-warning"
	}
	return status_class_names[status]
}

func (h Helpers) DisplayRunStatusButton(status scheduler.RunStatus) template.HTML {
	if status < scheduler.None || status > scheduler.Abandoned {
		return template.HTML(BTN_UNKNOWN)
	}
	return template.HTML(status_buttons[status])
}

func (h Helpers) DisplayAgo(from time.Time) string {
	if from.Equal(nilTime) {
		return "never"
	}
	return h.DisplayDuration(from, time.Now()) + " ago"
}

func (h Helpers) DisplayTime(t time.Time) string {
	return t.Format(time.RFC822)
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
		tm += fmt.Sprintf(" %dh", hours)
	}
	minutes := int(duration.Minutes()) % 60
	tm += fmt.Sprintf(" %dm", minutes)
	return tm
}
