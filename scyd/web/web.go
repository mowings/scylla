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

func getJobInfoJson(ctx *Context, name string, req *http.Request, r render.Render) {
	resp_chan := make(chan scheduler.StatusResponse)
	log.Println(name)
	status_req := scheduler.StatusRequest{Name: name, Chan: resp_chan}
	ctx.StatusChan <- status_req
	resp := <-resp_chan
	code := 200
	proto := req.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}

	if _, found := resp.(string); found == true {
		code = 404
	} else if job_list, found := resp.(*[]scheduler.JobReport); found == true {
		log.Println("Job report")
		for i, job := range *job_list { // Fill in detail link
			(*job_list)[i].DetailURI = fmt.Sprintf("%s://%s/api/v1/jobs/%s", proto, req.Host, job.Name)
		}
	} else if job_detail, found := resp.(*scheduler.JobReportWithHistory); found == true {
		job_detail.DetailURI = fmt.Sprintf("%s://%s/api/v1/jobs/%s", proto, req.Host, job_detail.Name)
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
		getJobInfoJson(ctx, "", req, r)
	})
	server.Get("/api/v1/jobs/:name", func(params martini.Params, req *http.Request, r render.Render) {
		getJobInfoJson(ctx, params["name"], req, r)
	})

	writeEndpoint(ctx.Config.Web.Listen)
	server.RunOnAddr(ctx.Config.Web.Listen)

}
