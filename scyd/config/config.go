package config

import (
	"code.google.com/p/gcfg"
	"errors"
	"fmt"
	"github.com/mowings/scylla/scyd/cronsched"
	"github.com/mowings/scylla/scyd/sched"
	"os"
	"path/filepath"
	"strings"
)

const DEFAULT_RUN_DIR = "/var/lib/scylla"
const DEFAULT_CONNECT_TIMEOUT = 20
const DEFAULT_RUN_TIMEOUT = 86400
const DEFAULT_MAX_RUN_HISTORY = 50

type PoolSpec struct {
	Name    string
	Host    []string
	Dynamic bool
}

type JobSpec struct {
	Name           string
	Command        []string
	Description    string
	Schedule       string
	ScheduleInst   sched.Sched `json:"-"`
	Keyfile        string
	Pass           string
	Host           string
	Pool           string
	PoolMode       string
	PoolInst       *PoolSpec `json:"-"`
	DefaultUser    string
	Upload         string
	Sudo           bool
	SudoCommand    string `gcfg:"sudo-command"`
	ConnectTimeout int    `gcfg:"connect-timeout"`
	RunTimeout     int    `gcfg:"run-timeout"`
	MaxRunHistory  int    `gcfg:"max-run-history"`
	RunOnStart     bool   `gcfg:"run-on-start"`
	Notifier       string
}

type Defaults struct {
	Keyfile        string
	Pass           string
	ConnectTimeout int    `gcfg:"connect-timeout"`
	RunTimeout     int    `gcfg:"run-timeout"`
	SudoCommand    string `gcfg:"sudo-command"`
	User           string
	Notify         string
	MaxRunHistory  int `gcfg:"max-run-history"`
}

type General struct {
	User string
}

type Notifier struct {
	Name        string
	Path        string
	Args        []string `gcfg:"arg"`
	EdgeTrigger bool     `gcfg:"edge-trigger"`
	NumFailures int      `gcfg:"num-failures"`
	Always      bool
}

type Web struct {
	Listen string
}

type Config struct {
	General  General
	Web      Web
	Defaults Defaults
	Pool     map[string]*PoolSpec
	Job      map[string]*JobSpec
	Notifier map[string]*Notifier
}

func New(fn string) (cfg *Config, err error) {
	var config = Config{}
	cfg = &config
	err = gcfg.ReadFileInto(cfg, fn)
	if err == nil {
		err = config.Validate()
	}
	if cfg.Defaults.ConnectTimeout == 0 {
		cfg.Defaults.ConnectTimeout = DEFAULT_CONNECT_TIMEOUT
	}
	if cfg.Defaults.RunTimeout == 0 {
		cfg.Defaults.RunTimeout = DEFAULT_RUN_TIMEOUT
	}
	if cfg.Defaults.MaxRunHistory == 0 {
		cfg.Defaults.MaxRunHistory = DEFAULT_MAX_RUN_HISTORY
	}

	// Cherry up pool hosts
	for name, pool := range cfg.Pool {
		pool.Name = name
	}

	for name, notifier := range cfg.Notifier {
		notifier.Name = name
		if notifier.Path == "" {
			return nil, errors.New(fmt.Sprintf("Notifier %s has no path set", name))
		}
		stat, err := os.Stat(notifier.Path)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Notifier %s -- cannot stat %s (%s)", name, notifier.Path, err.Error()))
		}
		if (stat.Mode() & 0111) == 0 {
			return nil, errors.New(fmt.Sprintf("Notifier %s -- %s must be executable", name, notifier.Path))
		}
	}

	// Parse the schedule data, set defaults
	for name, job := range cfg.Job {
		job.Name = name
		job.ParseSchedule()
		if err != nil {
			return nil, err
		}
		if job.ConnectTimeout == 0 {
			job.ConnectTimeout = cfg.Defaults.ConnectTimeout
		}
		if job.RunTimeout == 0 {
			job.RunTimeout = cfg.Defaults.RunTimeout
		}
		if job.MaxRunHistory == 0 {
			job.MaxRunHistory = cfg.Defaults.MaxRunHistory
		}
		if job.Keyfile == "" {
			job.Keyfile = cfg.Defaults.Keyfile
		}
		if job.Pass == "" {
			job.Pass = cfg.Defaults.Pass
		}
		job.DefaultUser = cfg.Defaults.User
		if job.Pool != "" {
			p := strings.Split(job.Pool, " ")
			if len(p) > 1 {
				job.PoolMode = p[1]
			}
			job.PoolInst = cfg.Pool[p[0]]
			if job.PoolInst == nil {
				return nil, errors.New(fmt.Sprintf("Bad pool %s specified by job %s", name, p[0]))
			}
		}
		if job.Notifier != "" && cfg.Notifier[job.Notifier] == nil {
			return nil, errors.New(fmt.Sprintf("Bad notifier %s specified by job %s", job.Notifier, name))
		}
	}

	return cfg, err
}

func (job *JobSpec) ParseSchedule() error {
	if job.Schedule == "" {
		job.ScheduleInst = &sched.NoSchedule{}
		return nil
	}
	m := sched.RexSched.FindStringSubmatch(job.Schedule)
	if m == nil {
		errors.New("Unable to parse schedule: " + job.Schedule)
	}
	if m[1] == "cron" {
		job.ScheduleInst = &cronsched.ParsedCronSched{}
	} else {
		return errors.New("Unknown schedule type: " + job.Schedule)
	}
	err := job.ScheduleInst.Parse(m[2])
	return err
}

func (cfg *Config) Validate() (err error) {
	return err
}

func RunDir() string {
	root := os.Getenv("SCYLLA_PATH")
	if root == "" {
		root = DEFAULT_RUN_DIR
	}
	return filepath.Join(root, "run")
}

func JobDir() string {
	return filepath.Join(RunDir(), "jobs")
}

func PoolCacheDir() string {
	return filepath.Join(RunDir(), "pools")
}
