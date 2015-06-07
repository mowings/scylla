package web

import (
	"github.com/martini-contrib/render"
	"net/http"
)

func renderJobInfoJson(ctx *Context, parts []string, req *http.Request, r render.Render) {
	code, resp := getJobInfo(ctx, parts, req, r)
	r.JSON(code, resp)
}
