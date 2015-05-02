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
	cfg_path := flag.String("config", "/etc/scylla.conf", "Config file path")
	flag.Parse()
	log.Printf("Starting scylla server...")
	cfg, err := config.New(*cfg_path)
	if err != nil {
		log.Fatal("Unable to parse config file: " + err.Error())
		os.Exit(-1)
	}
	_, _ = scheduler.Run()
	web.Run(cfg.Web.Listen)
}
