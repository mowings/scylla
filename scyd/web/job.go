package web

import (
	"fmt"
	"github.com/martini-contrib/render"
	"github.com/mowings/scylla/scyd/scheduler"
	"log"
	"net/http"
	"reflect"
)

func getJobInfo(ctx *Context, parts []string, req *http.Request, r render.Render) (int, scheduler.StatusResponse) {
	resp_chan := make(chan scheduler.StatusResponse)
	status_req := scheduler.StatusRequest{Object: parts, Chan: resp_chan}
	ctx.StatusChan <- status_req
	resp := <-resp_chan
	code := 200
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}

	// Annotate with detail URI references
	log.Printf("Scheduler response: %s\n", reflect.TypeOf(resp).String())
	if _, found := resp.(string); found == true {
		code = 404
	} else if job_list, found := resp.(*[]scheduler.JobReport); found == true {
		log.Println("Job report")
		for i, job := range *job_list { // Fill in detail link
			(*job_list)[i].DetailURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s", job.Name), req)
		}
	} else if job_detail, found := resp.(*scheduler.JobReportWithHistory); found == true {
		job_detail.DetailURI = fmt.Sprintf("%s://%s/api/v1/jobs/%s", proto, req.Host, job_detail.Name)
		for i, _ := range job_detail.Runs {
			job_detail.Runs[i].DetailURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d", job_detail.Name, job_detail.Runs[i].RunId), req)
		}
	} else if run, found := resp.(*scheduler.JobRun); found == true {
		run.DetailURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d", run.JobName, run.RunId), req)
		for i, hr := range run.HostRuns {
			for j, _ := range hr.CommandRuns {
				run.HostRuns[i].CommandRuns[j].StdOutURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d/%d/%d/stdout", run.JobName, run.RunId, hr.HostId, j), req)
				run.HostRuns[i].CommandRuns[j].StdErrURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d/%d/%d/stderr", run.JobName, run.RunId, hr.HostId, j), req)
			}
		}
	}
	return code, resp
}
