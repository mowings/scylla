package cronsched

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	var sched ParsedCronSched
	err := sched.Parse("15,0 */2 10-20,31 1 *")
	if err != nil {
		t.Error("Got error on parse " + err.Error())
	}
	fmt.Println(sched.Minutes)
	fmt.Println(sched.Hours)
	fmt.Println(sched.Mday)
	fmt.Println(sched.Month)
	fmt.Println(sched.Dow)
}
