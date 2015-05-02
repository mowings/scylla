package web

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func Run(addr string) {
	server := martini.Classic()
	server.Use(render.Renderer())
	server.Get("/", func() string {
		return "<h1>Scylla</h1>"
	})
	server.RunOnAddr(addr)

}
