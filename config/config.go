package config

import (
	"code.google.com/p/gcfg"
)

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
	Keyfile        string
	Password       string
	ConnectTimeout int    `gcfg:"connect-timeout"`
	RunTimeout     int    `gcfg:"run-timeout"`
	SudoCommand    string `gcfg:"sudo-command"`
	User           string
	Port           int
	OnFailure      string `gcfg:"on-failure"`
}

type Config struct {
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
	return cfg, err
}

func (cfg *Config) Validate() (err error) {

	return err
}
