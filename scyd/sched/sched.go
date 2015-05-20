package sched

import (
	"time"
)

type Sched interface {
	Unparsed() string
	Type() string
	Match(t *time.Time) bool
	Parse(line string) (err error)
}
