package cronsched

import (
	"errors"
	//	"log"
	"regexp"
	"strconv"
	"strings"
	//	"time"
)

// Regexps
var NUM_REX = regexp.MustCompile("^\\d{1,2}$")
var RANGE_REX = regexp.MustCompile("^\\d{1,2}-\\d{1,2}$")
var STEP_REX = regexp.MustCompile("^\\*/\\d{1,2}$")

type ParsedCronSched struct {
	Line    string
	Minutes []int
	Hours   []int
	Mday    []int
	Month   []int
	Dow     []int
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

func parseCronSection(section string, divs int, offset int) (units []int, err error) {
	units = make([]int, 0, divs)
	if section == "*" {
		for i := 0; i < cap(units); i++ {
			units = append(units, i+offset)
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
			units = appendWithoutDupes(units, unit)
		} else if RANGE_REX.MatchString(part) {
			units, err = processRange(part, units, offset)
		} else if STEP_REX.MatchString(part) {
			units, err = processSteps(part, units, offset)
		} else {
			return nil, errors.New("Did not understand stanza: " + part)
		}

	}
	return units, nil
}

func processRange(stanza string, units []int, offset int) ([]int, error) {
	parts := strings.Split(stanza, "-")
	start, _ := strconv.Atoi(parts[0])
	end, _ := strconv.Atoi(parts[1])
	if start >= cap(units)+offset || end >= cap(units)+offset {
		return nil, errors.New("stanza out of range: " + stanza)
	}
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		units = appendWithoutDupes(units, i)
	}
	return units, nil
}

func processSteps(stanza string, units []int, offset int) ([]int, error) {
	parts := strings.Split(stanza, "/")
	step, _ := strconv.Atoi(parts[1])
	if step >= cap(units)+offset {
		return nil, errors.New("stanza out of range: " + stanza)
	}
	for i := 0; i < cap(units); i += step {
		units = appendWithoutDupes(units, i+offset)
	}
	return units, nil
}

func appendWithoutDupes(units []int, unit int) []int {
	for _, u := range units {
		if u == unit {
			return units
		}
	}
	return append(units, unit)
}
