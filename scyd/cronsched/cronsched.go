package cronsched

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Regexps
var NUM_REX = regexp.MustCompile("^\\d{1,2}$")
var RANGE_REX = regexp.MustCompile("^\\d{1,2}-\\d{1,2}$")
var STEP_REX = regexp.MustCompile("^\\*/\\d{1,2}$")

type ParsedCronSched struct {
	unparsed string
	minutes  map[int]bool
	hours    map[int]bool
	mdays    map[int]bool
	months   map[int]bool
	dows     map[int]bool
}

func (sched *ParsedCronSched) Type() string {
	return "cron"
}

func (sched *ParsedCronSched) Unparsed() string {
	return sched.unparsed
}

func (sched *ParsedCronSched) Match(t *time.Time) bool {
	h, m, _ := t.Clock()
	_, mon, mday := t.Date()
	dow := t.Weekday()
	if sched.minutes[m] && sched.hours[h] && sched.mdays[mday] && sched.months[int(mon)] && sched.dows[int(dow)] {
		return true
	}

	return false
}

func (sched *ParsedCronSched) Parse(line string) (err error) {
	sched.unparsed = line

	parts := strings.Split(line, " ")

	if len(parts) != 5 {
		return errors.New("Wrong number of sections in cron entry: " + line)
	}

	if sched.minutes, err = parseCronSection(parts[0], 60, 0); err != nil {
		return err
	}
	if sched.hours, err = parseCronSection(parts[1], 24, 0); err != nil {
		return err
	}
	if sched.mdays, err = parseCronSection(parts[2], 31, 1); err != nil {
		return err
	}
	if sched.months, err = parseCronSection(parts[3], 12, 1); err != nil {
		return err
	}
	sched.dows, err = parseCronSection(parts[4], 7, 0)

	return err
}

func parseCronSection(section string, divs int, offset int) (units map[int]bool, err error) {
	units = make(map[int]bool)
	if section == "*" {
		for i := 0; i < divs; i++ {
			units[i+offset] = true
		}
		return units, nil // Match anything
	}
	parts := strings.Split(section, ",")
	for _, part := range parts {
		if NUM_REX.MatchString(part) {
			unit, _ := strconv.Atoi(part)
			if unit >= divs+offset {
				return nil, errors.New("Cron stanza " + part + " is out of range.")
			}
			units[unit] = true
		} else if RANGE_REX.MatchString(part) {
			err = processRange(part, units, divs, offset)
		} else if STEP_REX.MatchString(part) {
			err = processSteps(part, units, divs, offset)
		} else {
			return nil, errors.New("Did not understand stanza: " + part)
		}

	}
	return units, nil
}

func processRange(stanza string, units map[int]bool, divs int, offset int) error {
	parts := strings.Split(stanza, "-")
	start, _ := strconv.Atoi(parts[0])
	end, _ := strconv.Atoi(parts[1])
	if start >= divs+offset || end >= divs+offset {
		return errors.New("stanza out of range: " + stanza)
	}
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		(units)[i+offset] = true
	}
	return nil
}

func processSteps(stanza string, units map[int]bool, divs int, offset int) error {
	parts := strings.Split(stanza, "/")
	step, _ := strconv.Atoi(parts[1])
	if step >= divs+offset {
		return errors.New("stanza out of range: " + stanza)
	}
	for i := 0; i < divs; i += step {
		units[i+offset] = true
	}
	return nil
}
