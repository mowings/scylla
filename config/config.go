package config

import (
	"code.google.com/p/gcfg"
	"github.com/mowings/scylla/sched"
	"io/ioutil"
	"log"
	"os"
)

type RemoteSpec struct {
	HostConn string //user@host:[port]
	Password string
	Keyfile  string
}

type PoolSpec struct {
	Name    string
	Remotes []*RemoteSpec
}

type CommandSpec struct {
	Command        string
	Sudo           int
	RunTimemout    int
	ConnectTimeout int
}

type JobConfig struct {
	Command     string
	Description string
	Schedule    sched.Sched
	Remote      *RemoteSpec
	Pool        *PoolSpec
	Commands    []CommandSpec
}

type Defaults struct {
	Keyfile        string
	Password       string
	Connecttimeout int
	Runtimeout     int
}

type Config struct {
	Defaults Defaults
	Jobs     map[string]*JobConfig
}

func New(fn string) (cfg *Config, err error) {
	if _, err := os.Stat(fn); err != nil {
		panic("Unable to open config file " + fn + ": " + err.Error())
	}
	dat, _ := ioutil.ReadFile(fn)
	var config = Config{}
	cfg = &config
	err = gcfg.ReadStringInto(cfg, string(dat))
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}
	return cfg, err
}
