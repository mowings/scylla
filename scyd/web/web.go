package web

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/render"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/scheduler"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Context struct {
	CfgPath    string
	LoadChan   chan string
	StatusChan chan scheduler.StatusRequest
	Config     config.Config
}

func loadConfig(ctx Context) (*config.Config, error) {
	cfg, err := config.New(ctx.CfgPath)
	if err != nil {
		return nil, err
	}
	ctx.LoadChan <- ctx.CfgPath
	return cfg, nil
}

func validateConfig(ctx Context) (err error) {
	_, err = config.New(ctx.CfgPath)
	return err
}

func qualifyURL(path string, req *http.Request) string {
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}
	return fmt.Sprintf("%s://%s%s", proto, req.Host, path)
}

func renderJobInfoJson(ctx *Context, parts []string, req *http.Request, r render.Render) {
	code, resp := getJobInfo(ctx, parts, req, r)
	r.JSON(code, resp)
}

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
	var run_resp scheduler.StatusResponse
	var run *scheduler.RunHistoryReport
	var host_run *scheduler.HostRunReport
	code, run_resp = getJobInfo(ctx, []string{jobname, runid}, req, r)
	if code != 200 {
		r.HTML(code, "error", run_resp)
		return
	}
	run = run_resp.(*scheduler.RunHistoryReport)
	id, err := strconv.Atoi(hostid)
	if err != nil || run.GetHostRunById(id) == nil {
		r.HTML(404, "error", "Host ID not found")
		return
	} else {
		host_run = run.GetHostRunById(id)
	}
	dot := struct {
		Job     *scheduler.JobReportWithHistory
		Run     *scheduler.RunHistoryReport
		HostRun *scheduler.HostRunReport
		Helpers Helpers
	}{
		job,
		run,
		host_run,
		Helpers{},
	}
	r.HTML(code, "host", dot)
}

func sanitize(path string) string {
	clean := strings.Replace(path, "..", "", -1)
	return clean
}

func getJobOutput(jobname, jobid, host, command_id, fn string, res http.ResponseWriter) {
	res.Header().Set("Content-Type", "text/plain")
	path := sanitize(filepath.Join(config.RunDir(), jobname, jobid, host, command_id, fn))
	log.Println(path)
	r, err := os.Open(path)
	if err == nil {
		defer r.Close()
		_, err = io.Copy(res, r)
	} else {
		http.Error(res, "Not Found", http.StatusNotFound)
	}

}

func writeEndpoint(endpoint string) {
	err := ioutil.WriteFile("/var/run/scylla.endpoint", []byte(endpoint), 0644)
	if err != nil {
		panic(err)
	}
}

func Run(ctx *Context) {
	loadConfig(*ctx) // Force a load on startup
	server := martini.Classic()
	server.Use(gzip.All())
	server.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	server.Get("/", func(r render.Render) {
		r.Redirect("/jobs", 302)
	})
	server.Get("/jobs", func(req *http.Request, r render.Render) {
		renderJobListHtml(ctx, req, r)
	})
	server.Get("/jobs/:name", func(params martini.Params, req *http.Request, r render.Render) {
		renderJobDetailHtml(params["name"], ctx, req, r)
	})
	server.Get("/jobs/:name/:runid/:hostid", func(params martini.Params, req *http.Request, r render.Render) {
		renderHostDetailHtml(params["name"], params["runid"], params["hostid"], ctx, req, r)
	})

	server.Put("/api/v1/reload", func(req *http.Request, r render.Render) {
		loadConfig(*ctx)
	})
	server.Get("/api/v1/test", func(req *http.Request, r render.Render) {
		validateConfig(*ctx)
	})
	server.Get("/api/v1/jobs", func(req *http.Request, r render.Render) {
		renderJobInfoJson(ctx, []string{}, req, r)
	})
	server.Get("/api/v1/jobs/:name", func(params martini.Params, req *http.Request, r render.Render) {
		renderJobInfoJson(ctx, []string{params["name"]}, req, r)
	})
	server.Get("/api/v1/jobs/:name/:id", func(params martini.Params, req *http.Request, r render.Render) {
		renderJobInfoJson(ctx, []string{params["name"], params["id"]}, req, r)
	})
	server.Get("/api/v1/jobs/:name/:id/:host_id/:command_id/:fn", func(params martini.Params, res http.ResponseWriter) {
		getJobOutput(params["name"], params["id"], params["host_id"], params["command_id"], params["fn"], res)
	})

	writeEndpoint(ctx.Config.Web.Listen)
	server.RunOnAddr(ctx.Config.Web.Listen)

}
