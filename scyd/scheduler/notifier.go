package scheduler

import (
	"github.com/mowings/scylla/scyd/config"
	"log"
	"os/exec"
	"strconv"
)

type JobNotifier struct {
	config.Notifier
}

func (notifier JobNotifier) Notify(job *Job) {
	last_status := Succeeded
	if len(job.History) >= 2 {
		last_status = job.History[1].Status
	}
	if notifier.EdgeTrigger && job.Status != last_status {
		notifier.fireNotification(job)
	} else if notifier.Always {
		notifier.fireNotification(job)
	} else if job.Status == Succeeded && last_status == Failed {
		notifier.fireNotification(job)
	} else if job.Status == Failed {
		notifier.fireNotification(job)
	}
}

func (notifier JobNotifier) fireNotification(job *Job) {
	args := make([]string, 3)
	args[0] = RunStatusNames[job.Status]
	args[1] = job.Name
	args[2] = strconv.Itoa(job.History[0].RunId)
	args = append(args, notifier.Args...)
	cmd := exec.Command(notifier.Path, args...)
	log.Printf("Firing notification command %s %v", notifier.Path, args)
	go cmd.Run()
}
