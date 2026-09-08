package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host            string `yaml:"host"`
	Port            string `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	PrivateKeyPath  string `yaml:"private_key_path"`
}

type Config struct {
	Server                ServerConfig `yaml:"server"`
	CheckIntervalSeconds  int          `yaml:"check_interval_seconds"`
	WebPort               string       `yaml:"web_port"`
	DatabasePath          string       `yaml:"database_path"`
}

// Load membaca file config.yaml dan mengembalikan struct Config
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// default value kalau tidak diisi di yaml
	if cfg.CheckIntervalSeconds == 0 {
		cfg.CheckIntervalSeconds = 30
	}
	if cfg.WebPort == "" {
		cfg.WebPort = "8080"
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "./data/monitoring.db"
	}

	return &cfg, nil
}
