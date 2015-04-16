package cronsched

import (
	"errors"
	//	"log"
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
	Line    string
	Minutes map[int]bool
	Hours   map[int]bool
	Mday    map[int]bool
	Month   map[int]bool
	Dow     map[int]bool
}

func (sched *ParsedCronSched) Match(t *time.Time) bool {
	h, m, _ := t.Clock()
	_, mon, mday := t.Date()
	dow := t.Weekday()

	if sched.Minutes[m] && sched.Hours[h] && sched.Mday[mday] && sched.Month[int(mon)] && sched.Dow[int(dow)] {
		return true
	}

	return false
}

func (sched *ParsedCronSched) Parse(line string) (err error) {
	parts := strings.Split(line, " ")

	if len(parts) != 5 {
		return errors.New("Wrong number of sections in cron entry")
	}
	sched.Line = line

	if sched.Minutes, err = parseCronSection(parts[0], 60, 0); err != nil {
		return err
	}
	if sched.Hours, err = parseCronSection(parts[1], 24, 0); err != nil {
		return err
	}
	if sched.Mday, err = parseCronSection(parts[2], 31, 1); err != nil {
		return err
	}
	if sched.Month, err = parseCronSection(parts[3], 12, 1); err != nil {
		return err
	}
	sched.Dow, err = parseCronSection(parts[4], 7, 0)

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
