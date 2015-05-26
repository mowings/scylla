package web

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/scheduler"
	"io/ioutil"
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

func getJobListJson(ctx *Context, req *http.Request, r render.Render) {
	resp_chan := make(chan scheduler.StatusResponse)
	status_req := scheduler.StatusRequest{Name: "", Chan: resp_chan}
	ctx.StatusChan <- status_req
	resp := <-resp_chan
	r.JSON(200, resp)
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
	server.Get("/api/v1/jobs.json", func(req *http.Request, r render.Render) {
		getJobListJson(ctx, req, r)
	})

	writeEndpoint(ctx.Config.Web.Listen)
	server.RunOnAddr(ctx.Config.Web.Listen)

}
