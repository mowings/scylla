package web

import (
	"fmt"
	"github.com/mowings/scylla/scyd/scheduler"
	"html/template"
	"time"
)

var nilTime = time.Time{}

var status_buttons = []string{
	"<button class=\"btn btn-status btn-small btn-info\">none</button>",
	"<button class=\"btn btn-status btn-small btn-info\">running</button>",
	"<button class=\"btn btn-status btn-small btn-success\">success</button>",
	"<button class=\"btn btn-status btn-small btn-danger\">failed</button>",
	"<button class=\"btn btn-status btn-small btn-danger\">cancelled</button>",
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
	if to.Equal(nilTime) {
		duration = time.Now().Sub(from) // Assume duration is for a running job with a nil to time
	}
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
