package web

import (
	"github.com/martini-contrib/render"
	"net/http"
	"scyd/scheduler"
	"strconv"
)

func renderJobListHtml(ctx *Context, req *http.Request, r render.Render) {
	code, resp := getJobInfo(ctx, []string{}, req, r)
	joblist := resp.(*[]scheduler.JobReport)
	dot := struct {
		Jobs    *[]scheduler.JobReport
		Helpers Helpers
	}{
		joblist,
		Helpers{},
	}
	r.HTML(code, "jobs", dot)
}

func renderJobDetailHtml(name string, ctx *Context, req *http.Request, r render.Render) {
	code, resp := getJobInfo(ctx, []string{name}, req, r)
	if code != 200 {
		r.HTML(code, "error", resp)
		return
	}
	job := resp.(*scheduler.JobReportWithHistory)
	dot := struct {
		Job     *scheduler.JobReportWithHistory
		Helpers Helpers
	}{
		job,
		Helpers{},
	}
	r.HTML(code, "job", dot)
}

func renderHostDetailHtml(jobname string, runid string, hostid string, ctx *Context, req *http.Request, r render.Render) {
	code, resp := getJobInfo(ctx, []string{jobname}, req, r)
	if code != 200 {
		r.HTML(code, "error", resp)
		return
	}
	job := resp.(*scheduler.JobReportWithHistory)
	job.Runs = nil // Free up that memory
	code, run_resp := getJobInfo(ctx, []string{jobname, runid}, req, r)
	if code != 200 {
		r.HTML(code, "error", run_resp)
		return
	}
	run := run_resp.(*scheduler.JobRun)
	id, err := strconv.Atoi(hostid)
	if err != nil || run.GetHostRunById(id) == nil {
		r.HTML(404, "error", "Host ID not found")
		return
	}
	host_run := run.GetHostRunById(id)
	dot := struct {
		Job     *scheduler.JobReportWithHistory
		Run     *scheduler.JobRun
		HostRun *scheduler.HostRun
		Helpers Helpers
	}{
		job,
		run,
		host_run,
		Helpers{},
	}
	r.HTML(code, "host", dot)
}
