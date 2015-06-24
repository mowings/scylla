package web

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/render"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/scheduler"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Context struct {
	CfgPath string
	ReqChan chan scheduler.Request
	Config  config.Config
}

func loadConfig(ctx Context) (*config.Config, error) {
	cfg, err := config.New(ctx.CfgPath)
	if err != nil {
		return nil, err
	}
	ctx.ReqChan <- scheduler.LoadConfigRequest(ctx.CfgPath)
	return cfg, nil
}

func validateConfig(ctx Context) (err error) {
	_, err = config.New(ctx.CfgPath)
	return err
}

func sanitize(path string) string {
	clean := strings.Replace(path, "..", "", -1)
	return clean
}

func getJobOutput(jobname, jobid, host, command_id, fn string, res http.ResponseWriter) {
	res.Header().Set("Content-Type", "text/plain")
	path := sanitize(filepath.Join(config.JobDir(), jobname, jobid, host, command_id, fn))
	log.Println(path)
	r, err := os.Open(path)
	if err == nil {
		defer r.Close()
		_, err = io.Copy(res, r)
	} else {
		http.Error(res, "Not Found", http.StatusNotFound)
	}

}

func Run(ctx *Context) {
	loadConfig(*ctx) // Force a load on startup
	logger := log.New(os.Stdout, "[martini] ", log.Ldate|log.Ltime)
	server := martini.Classic()
	server.Map(logger)
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
		_, err := loadConfig(*ctx)
		if err != nil {
			r.JSON(400, err.Error())
		} else {
			r.JSON(200, "ok")
		}
	})
	server.Put("/api/v1/run/:job", func(params martini.Params, req *http.Request, r render.Render) {
		ctx.ReqChan <- scheduler.RunJobRequest(params["job"])
	})

	server.Put("/api/v1/fail/:job", func(params martini.Params, req *http.Request, r render.Render) {
		change_run_status_req := scheduler.ChangeJobStatusRequest{Name: params["job"], Status: scheduler.Failed}
		ctx.ReqChan <- change_run_status_req
	})

	server.Put("/api/v1/pool/:pool", func(params martini.Params, req *http.Request, r render.Render) {
		updatePool(ctx, params["pool"], req, r)
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

	server.RunOnAddr(ctx.Config.Web.Listen)

}
