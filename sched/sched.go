package sched

import (
	"time"
)

type Sched interface {
	Match(t *time.Time) bool
	Parse(line string) (err error)
}
