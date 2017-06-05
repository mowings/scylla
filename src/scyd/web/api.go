package web

import (
	"encoding/json"
	"github.com/martini-contrib/render"
	"net/http"
	"scyd/scheduler"
)

func renderJobInfoJson(ctx *Context, parts []string, req *http.Request, r render.Render) {
	code, resp := getJobInfo(ctx, parts, req, r)
	r.JSON(code, resp)
}

func updatePool(ctx *Context, pool_name string, req *http.Request, r render.Render) {
	var hosts []string
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&hosts)
	if err != nil {
		r.JSON(400, err.Error())
	} else {
		pr := scheduler.UpdatePoolRequest{Name: pool_name, Hosts: hosts}
		ctx.ReqChan <- pr
		r.JSON(200, "ok")
	}
}
