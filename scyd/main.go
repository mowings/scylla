package main

import (
	"flag"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/scheduler"
	"github.com/mowings/scylla/scyd/web"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func writePid() {
	pid := os.Getpid()
	err := ioutil.WriteFile("/var/run/scylla.pid", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		panic(err)
	}
}

func writeEndpoint(endpoint string) {
	err := ioutil.WriteFile("/var/run/scylla.endpoint", []byte(endpoint), 0644)
	if err != nil {
		panic(err)
	}
}

func setUser(name string) (err error) {
	user, err := user.Lookup(name)
	if err == nil {
		nid, _ := strconv.Atoi(user.Uid)
		err = syscall.Setuid(nid)
	}
	return err
}

func main() {
	var ctx web.Context
	cfg_path := flag.String("config", "/etc/scylla.conf", "Config file path")
	flag.Parse()
	log.Printf("Starting scylla server...")

	ctx.CfgPath = *cfg_path
	// Vaildate config on startup and save pid and endpoints
	cfg, err := config.New(ctx.CfgPath)
	if err != nil {
		log.Printf("Unable to load config: %s", err.Error())
		os.Exit(-1)
	}

	writePid()
	writeEndpoint(cfg.Web.Listen)
	if cfg.General.User != "" {
		log.Printf("Running as : %s", cfg.General.User)
		if err := setUser(cfg.General.User); err != nil {
			log.Printf("Unable to change uid: %s", err.Error())
			os.Exit(-1)
		}
	}

	ctx.ReqChan = scheduler.Run()
	ctx.Config = *cfg
	web.Run(&ctx)
}
