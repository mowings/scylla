package config

import (
	"log"
	"testing"
)

func TestParse(t *testing.T) {
	config, err := New("test.ini")
	if err != nil {
		t.Error("Got error on parse " + err.Error())
	} else {
		log.Println(*config)
		log.Println(*config.Jobs["test"])
	}
}
