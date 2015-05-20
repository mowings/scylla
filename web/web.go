package web

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/mowings/scylla/config"
	"net/http"
)

type Context struct {
	CfgPath  string
	LoadChan chan string
	Config   config.Config
}

func loadConfig(ctx Context) (*config.Config, error) {
	cfg, err := config.New(ctx.CfgPath)
	if err != nil {
		return nil, err
	}
	ctx.LoadChan <- ctx.CfgPath
	return cfg, nil
}

func Run(ctx *Context) {
	server := martini.Classic()
	server.Use(render.Renderer())
	server.Get("/", func() string {
		return "<h1>Scylla</h1>"
	})
	server.Put("/api/v1/config", func(req *http.Request, r render.Render) {
		loadConfig(*ctx)
	})

	server.RunOnAddr(ctx.Config.Web.Listen)

}
