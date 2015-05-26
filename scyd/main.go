package main

import (
	"flag"
	"github.com/mowings/scylla/scyd/config"
	"github.com/mowings/scylla/scyd/scheduler"
	"github.com/mowings/scylla/scyd/web"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func writePid() {
	pid := os.Getpid()
	err := ioutil.WriteFile("/var/run/scylla.pid", []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		panic(err)
	}
}

func main() {
	var ctx web.Context
	cfg_path := flag.String("config", "/etc/scylla.conf", "Config file path")
	flag.Parse()
	log.Printf("Starting scylla server...")
	ctx.CfgPath = *cfg_path
	ctx.LoadChan, ctx.StatusChan = scheduler.Run()
	cfg, err := config.New(ctx.CfgPath)
	if err != nil {
		log.Fatal("Unable to parse config file: " + err.Error())
		os.Exit(-1)
	}
	writePid()
	ctx.Config = *cfg
	web.Run(&ctx)
}
