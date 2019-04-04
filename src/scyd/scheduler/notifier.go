package scheduler

import (
	"log"
	"os/exec"
	"scyd/config"
	"strconv"
)

type JobNotifier struct {
	config.Notifier
}

func (notifier JobNotifier) Notify(job *Job) {
	if notifier.Name == "none" {
		return
	}
	last_status := Succeeded
	if len(job.History) >= 2 {
		last_status = job.History[1].Status
	}
	cf := job.consecutiveFailures()
	log.Printf("consecutive failures: %d", cf)
	if notifier.EdgeTrigger {
		if job.Status != last_status {
			notifier.fireNotification(job)
		}
	} else if notifier.Always {
		notifier.fireNotification(job)
	} else if job.Status == Succeeded && last_status == Failed {
		if cf >= job.FailsToNotify {
			notifier.fireNotification(job)
		}
	} else if job.Status == Failed && (cf == job.FailsToNotify || job.FailsToNotify == 0) {
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
