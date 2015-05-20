package main

import (
	"flag"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/scheduler"
	"github.com/mowings/scylla/web"
	"log"
	"os"
)

func main() {
	var ctx web.Context
	cfg_path := flag.String("config", "/etc/scylla.conf", "Config file path")
	flag.Parse()
	log.Printf("Starting scylla server...")
	ctx.CfgPath = *cfg_path
	ctx.LoadChan, _ = scheduler.Run()
	cfg, err := config.New(ctx.CfgPath)
	if err != nil {
		log.Fatal("Unable to parse config file: " + err.Error())
		os.Exit(-1)
	}
	ctx.Config = *cfg
	web.Run(&ctx)
}
