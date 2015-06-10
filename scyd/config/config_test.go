package config

import (
	"log"
	"testing"
)

func TestParse(t *testing.T) {
	cfg, err := New("test.conf")
	if err != nil {
		t.Error("Got error on parse " + err.Error())
	} else {
		log.Printf("General: %+v\n", cfg.General)
		log.Printf("Web: %+v\n", cfg.Web)
		log.Printf("Defaults: %+v\n", cfg.Defaults)
		log.Println("Pools:")
		for k, v := range cfg.Pool {
			log.Printf("%s ==> %+v\n", k, *v)
		}
		log.Println("Jobs: ")
		for k, v := range cfg.Job {
			log.Printf("%s ==> %+v\n", k, *v)
			log.Printf("       %s (%s)\n", v.ScheduleInst.Type(), v.ScheduleInst.Unparsed())
		}
	}
}
