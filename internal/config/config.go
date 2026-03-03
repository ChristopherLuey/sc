package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	SSH         SSHConfig     `toml:"ssh"`
	Cluster     ClusterConfig `toml:"cluster"`
	JobDefaults JobDefaults   `toml:"job_defaults"`
}

type SSHConfig struct {
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	User         string `toml:"user"`
	IdentityFile string `toml:"identity_file"`
	UseAgent     bool   `toml:"use_agent"`
}

type ClusterConfig struct {
	DefaultPartition string        `toml:"default_partition"`
	RefreshInterval  time.Duration `toml:"-"`
	RefreshRaw       string        `toml:"refresh_interval"`
}

type JobDefaults struct {
	Account     string `toml:"account"`
	CPUsPerTask int    `toml:"cpus_per_task"`
	Memory      string `toml:"memory"`
	TimeLimit   string `toml:"time_limit"`
}

func Defaults() *Config {
	return &Config{
		SSH: SSHConfig{
			Host:     "sc.stanford.edu",
			Port:     22,
			UseAgent: true,
		},
		Cluster: ClusterConfig{
			DefaultPartition: "viscam",
			RefreshInterval:  10 * time.Second,
			RefreshRaw:       "10s",
		},
		JobDefaults: JobDefaults{
			Account:     "viscam",
			CPUsPerTask: 4,
			Memory:      "32G",
			TimeLimit:   "24:00:00",
		},
	}
}

// Load reads config from path. Returns (config, firstRun, error).
func Load(path string) (*Config, bool, error) {
	if path == "" {
		path = DefaultPath()
	}

	cfg := Defaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, true, nil
	}
	if err != nil {
		return nil, false, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, false, err
	}

	if cfg.Cluster.RefreshRaw != "" {
		d, err := time.ParseDuration(cfg.Cluster.RefreshRaw)
		if err == nil {
			cfg.Cluster.RefreshInterval = d
		}
	}

	return cfg, false, nil
}

func Save(cfg *Config, path string) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
