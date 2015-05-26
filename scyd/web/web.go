package web

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/scheduler"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
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

func getJobInfoJson(ctx *Context, parts []string, req *http.Request, r render.Render) {
	resp_chan := make(chan scheduler.StatusResponse)
	status_req := scheduler.StatusRequest{Object: parts, Chan: resp_chan}
	ctx.StatusChan <- status_req
	resp := <-resp_chan
	code := 200
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}
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
	} else if run, found := resp.(*scheduler.RunHistoryReport); found == true {
		run.DetailURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d", run.JobName, run.RunId), req)
		for i, hr := range run.HostRuns {
			for j, _ := range hr.CommandRuns {
				run.HostRuns[i].CommandRuns[j].StdOutURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d/%s/%d/stdout", run.JobName, run.RunId, hr.Host, j), req)
				run.HostRuns[i].CommandRuns[j].StdErrURI = qualifyURL(fmt.Sprintf("/api/v1/jobs/%s/%d/%s/%d/stderr", run.JobName, run.RunId, hr.Host, j), req)
			}
		}
	}

	r.JSON(code, resp)
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
	server.Use(render.Renderer())
	server.Get("/", func() string {
		return "<h1>Scylla</h1>"
	})
	server.Put("/api/v1/reload", func(req *http.Request, r render.Render) {
		loadConfig(*ctx)
	})
	server.Get("/api/v1/test", func(req *http.Request, r render.Render) {
		validateConfig(*ctx)
	})
	server.Get("/api/v1/jobs", func(req *http.Request, r render.Render) {
		getJobInfoJson(ctx, []string{}, req, r)
	})
	server.Get("/api/v1/jobs/:name", func(params martini.Params, req *http.Request, r render.Render) {
		getJobInfoJson(ctx, []string{params["name"]}, req, r)
	})
	server.Get("/api/v1/jobs/:name/:id", func(params martini.Params, req *http.Request, r render.Render) {
		getJobInfoJson(ctx, []string{params["name"], params["id"]}, req, r)
	})

	writeEndpoint(ctx.Config.Web.Listen)
	server.RunOnAddr(ctx.Config.Web.Listen)

}
