package config

import (
	"code.google.com/p/gcfg"
	"errors"
	"fmt"
	"github.com/mowings/scylla/scyd/cronsched"
	"github.com/mowings/scylla/scyd/sched"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const DEFAULT_RUN_DIR = "/var/scylla"
const DEFAULT_CONNECT_TIMEOUT = 20
const DEFAULT_RUN_TIMEOUT = 86400
const DEFAULT_MAX_RUN_HISTORY = 50

var host_parse = regexp.MustCompile(`^((?P<user>.+)@)?(?P<hostname>[^:]+)(:(?P<port>\d+))?`)

type PoolSpec struct {
	Name string
	Host []string
	File string
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
	Upload         string
	Sudo           bool
	SudoCommand    string `gcfg:"sudo-command"`
	ConnectTimeout int    `gcfg:"connect-timeout"`
	RunTimeout     int    `gcfg:"run-timeout"`
	MaxRunHistory  int    `gcfg:"max-run-history"`
}

type Defaults struct {
	Keyfile        string
	Pass           string
	ConnectTimeout int    `gcfg:"connect-timeout"`
	RunTimeout     int    `gcfg:"run-timeout"`
	SudoCommand    string `gcfg:"sudo-command"`
	User           string
	Port           int
	OnFailure      string `gcfg:"on-failure"`
	MaxRunHistory  int    `gcfg:"max-run-history"`
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
	if cfg.Defaults.ConnectTimeout == 0 {
		cfg.Defaults.ConnectTimeout = DEFAULT_CONNECT_TIMEOUT
	}
	if cfg.Defaults.RunTimeout == 0 {
		cfg.Defaults.RunTimeout = DEFAULT_RUN_TIMEOUT
	}
	if cfg.Defaults.MaxRunHistory == 0 {
		cfg.Defaults.MaxRunHistory = DEFAULT_MAX_RUN_HISTORY
	}

	// Qualify Pool hosts
	for _, pool := range cfg.Pool {
		for idx, host := range pool.Host {
			pool.Host[idx] = qualifyHost(host, cfg.Defaults.User, cfg.Defaults.Port)
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
		if job.Host != "" {
			job.Host = qualifyHost(job.Host, cfg.Defaults.User, cfg.Defaults.Port)
		}
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

func qualifyHost(unqualified string, default_user string, default_port int) (qualified string) {
	m := FindNamedStringCaptures(host_parse, unqualified)

	host := m["hostname"]
	user := m["user"]
	port := m["port"]

	if user == "" {
		user = default_user
	}
	if port == "" {
		port = strconv.Itoa(default_port)
	}

	return fmt.Sprintf("%s@%s:%s", user, host, port)

}

func FindNamedStringCaptures(re *regexp.Regexp, x string) map[string]string {
	matches := make(map[string]string)
	parts := re.FindStringSubmatch(x)
	if parts == nil {
		return nil
	}

	for index, key := range host_parse.SubexpNames() {
		if key != "" {
			matches[key] = parts[index]
		}
	}
	return matches
}

func RunDir() string {
	path := os.Getenv("SCYLLA_PATH")
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "./"
	}
	if path == "" {
		path = filepath.Join(cwd, "run")
	} else {
		path = filepath.Join(path, "run")
	}
	return path
}
