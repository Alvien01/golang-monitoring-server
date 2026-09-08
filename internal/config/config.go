package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host                  string `yaml:"host"`
	Port                  string `yaml:"port"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	PrivateKeyPath        string `yaml:"private_key_path"`
	PrivateKeyPassphrase  string `yaml:"private_key_passphrase"`
	KnownHostsPath        string `yaml:"known_hosts_path"`
	InsecureIgnoreHostKey bool   `yaml:"insecure_ignore_host_key"`
}

type Config struct {
	Server               ServerConfig `yaml:"server"`
	CheckIntervalSeconds int          `yaml:"check_interval_seconds"`
	WebPort              string       `yaml:"web_port"`
	DatabasePath         string       `yaml:"database_path"`
}

// Load membaca file config.yaml dan mengembalikan struct Config
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Dukungan ekspansi environment variable seperti ${SSH_PASSWORD}
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}

	// Fallback ke environment variable jika tidak diisi di YAML
	if cfg.Server.Password == "" {
		if envPass := os.Getenv("SSH_PASSWORD"); envPass != "" {
			cfg.Server.Password = envPass
		}
	}
	if cfg.Server.PrivateKeyPath == "" {
		if envKey := os.Getenv("SSH_PRIVATE_KEY_PATH"); envKey != "" {
			cfg.Server.PrivateKeyPath = envKey
		}
	}
	if cfg.Server.PrivateKeyPassphrase == "" {
		if envPassphrase := os.Getenv("SSH_PRIVATE_KEY_PASSPHRASE"); envPassphrase != "" {
			cfg.Server.PrivateKeyPassphrase = envPassphrase
		}
	}
	if cfg.Server.KnownHostsPath == "" {
		if envKnownHosts := os.Getenv("SSH_KNOWN_HOSTS_PATH"); envKnownHosts != "" {
			cfg.Server.KnownHostsPath = envKnownHosts
		}
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
