package main

import (
	"flag"
	"github.com/mowings/scylla/config"
	"github.com/mowings/scylla/scheduler"
	"github.com/mowings/scylla/web"
	"log"
	"os"
)

func loadConfig(path string, load_chan chan string) (*config.Config, error) {
	cfg, err := config.New(path)
	if err != nil {
		return nil, err
	}
	load_chan <- path
	return cfg, nil

}

func main() {
	cfg_path := flag.String("config", "/etc/scylla.conf", "Config file path")
	flag.Parse()
	log.Printf("Starting scylla server...")
	load_chan, _ := scheduler.Run()
	cfg, err := loadConfig(*cfg_path, load_chan)
	if err != nil {
		log.Fatal("Unable to parse config file: " + err.Error())
		os.Exit(-1)
	}
	web.Run(cfg.Web.Listen)
}
