package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"scyd/config"
	"scyd/scheduler"
	"scyd/web"
	"strconv"
	"syscall"
)

const VERSION = "1.01"

func writePid() {
	pid := os.Getpid()
	os.MkdirAll("/var/run/scylla", 0755)
	err := ioutil.WriteFile("/var/run/scylla/pid", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		panic(err)
	}
}

func writeEndpoint(endpoint string) {
	os.MkdirAll("/var/run/scylla", 0755)
	err := ioutil.WriteFile("/var/run/scylla/endpoint", []byte(endpoint), 0644)
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
	log.Printf("Starting scylla server v%s...", VERSION)

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
			log.Printf("WARNING: Unable to drop privileges: %s", err.Error())
		}
	}

	ctx.ReqChan = scheduler.Run()
	ctx.Config = *cfg
	web.Run(&ctx)
}
