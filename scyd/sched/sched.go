package sched

import (
	"regexp"
	"time"
)

var RexSched = regexp.MustCompile("^([a-z]+) (.+)")

type Sched interface {
	Unparsed() string
	Type() string
	Match(t *time.Time) bool
	Parse(line string) (err error)
}

type NoSchedule struct {
}

func (sched *NoSchedule) Unparsed() string {
	return ""
}

func (sched *NoSchedule) Type() string {
	return "none"
}

func (sched *NoSchedule) Match(t *time.Time) bool {
	return false
}

func (sched *NoSchedule) Parse(line string) error {
	return nil
}
