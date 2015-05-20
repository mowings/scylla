package config

import (
	"code.google.com/p/gcfg"
)

const DEFAULT_RUN_DIR = "/var/scylla"

type PoolSpec struct {
	Name string
	Host []string
	File string
}

type JobSpec struct {
	Command        []string
	Description    string
	Schedule       string
	Host           string
	Pool           string
	Upload         string
	Sudo           bool
	SudoCommand    string `gcfg:"sudo-command"`
	Manual         bool
	ConnectTimeout int `gcfg:"connect-timeout"`
	RunTimeout     int `gcfg:"run-timeout"`
}

type Defaults struct {
	RunDir         string `gcfg:"run-dir"`
	Keyfile        string
	Password       string
	ConnectTimeout int    `gcfg:"connect-timeout"`
	RunTimeout     int    `gcfg:"run-timeout"`
	SudoCommand    string `gcfg:"sudo-command"`
	User           string
	Port           int
	OnFailure      string `gcfg:"on-failure"`
}

type Web struct {
	Listen string
}

type Config struct {
	Web      Web
	Defaults Defaults
	Pool     map[string]*PoolSpec
	Job      map[string]*JobSpec
}

func New(fn string) (cfg *Config, err error) {
	var config = Config{}
	cfg = &config
	err = gcfg.ReadFileInto(cfg, fn)
	if err == nil {
		err = config.Validate()
	}
	if cfg.Defaults.RunDir == "" {
		cfg.Defaults.RunDir = DEFAULT_RUN_DIR
	}

	return cfg, err
}

func (cfg *Config) Validate() (err error) {

	return err
}
