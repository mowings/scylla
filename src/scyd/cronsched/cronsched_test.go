package cronsched

import (
	"fmt"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	var sched ParsedCronSched
	err := sched.Parse("15,0 */2 10-20,31 1 *")
	if err != nil {
		t.Error("Got error on parse " + err.Error())
	}
}

func TestMatch(t *testing.T) {
	var sched ParsedCronSched
	err := sched.Parse("* * * * *")
	if err != nil {
		t.Error("Got error on parse " + err.Error())
	}
	tm := time.Now()
	fmt.Println(tm.String())
	if !sched.Match(&tm) {
		t.Error("Failed match")
	}
	err = sched.Parse("10 3,15   * * *")
	tm, _ = time.Parse(time.RFC3339, "2014-11-26T03:15:00Z")
	if sched.Match(&tm) {
		t.Error("Matched when it shouldn't")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-26T03:10:00Z")
	if !sched.Match(&tm) {
		t.Error("Matched failed")
	}
	err = sched.Parse("*/10 3,15 * * *")
	if !sched.Match(&tm) {
		t.Error("Matched failed")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-26T03:18:00Z")
	if sched.Match(&tm) {
		t.Error("Matched when it shouldn't")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-26T15:30:00Z")
	err = sched.Parse("*/10 3,15 * * 3,5")
	if !sched.Match(&tm) {
		t.Error("Matched failed")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-27T15:30:00Z")
	if sched.Match(&tm) {
		t.Error("Matched when it shouldn't")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-26T15:30:00Z")
	err = sched.Parse("*/10 3,15 * * 3-5")
	if !sched.Match(&tm) {
		t.Error("Matched failed")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-27T15:30:00Z")
	if !sched.Match(&tm) {
		t.Error("Match failed")
	}
	tm, _ = time.Parse(time.RFC3339, "2014-11-29T15:30:00Z")
	if sched.Match(&tm) {
		t.Error("Match when it shouldn't")
	}

}
