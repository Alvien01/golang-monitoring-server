package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	ID                    string `yaml:"id"`
	Name                  string `yaml:"name"`
	Host                  string `yaml:"host"`
	Port                  string `yaml:"port"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	PrivateKeyPath        string `yaml:"private_key_path"`
	PrivateKeyPassphrase  string `yaml:"private_key_passphrase"`
	KnownHostsPath        string `yaml:"known_hosts_path"`
	InsecureIgnoreHostKey bool   `yaml:"insecure_ignore_host_key"`
}

type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Config struct {
	Servers              []ServerConfig `yaml:"servers"`
	Server               ServerConfig   `yaml:"server"` // Kompatibilitas mundur konfigurasi single server
	Auth                 AuthConfig     `yaml:"auth"`
	RetentionDays        int            `yaml:"retention_days"`
	CheckIntervalSeconds int            `yaml:"check_interval_seconds"`
	WebPort              string         `yaml:"web_port"`
	DatabasePath         string         `yaml:"database_path"`
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

	// Kompatibilitas mundur: jika 'servers' kosong tetapi 'server' ada
	if len(cfg.Servers) == 0 && cfg.Server.Host != "" {
		cfg.Servers = append(cfg.Servers, cfg.Server)
	}

	// Normalisasi dan fallback environment variable untuk setiap server
	for i := range cfg.Servers {
		s := &cfg.Servers[i]
		if s.ID == "" {
			if s.Host != "" {
				s.ID = s.Host
			} else {
				s.ID = "default"
			}
		}
		if s.Name == "" {
			s.Name = s.ID
		}
		if s.Port == "" {
			s.Port = "22"
		}

		// Fallback environment variable jika hanya 1 server dan belum diisi
		if len(cfg.Servers) == 1 {
			if s.Password == "" {
				if envPass := os.Getenv("SSH_PASSWORD"); envPass != "" {
					s.Password = envPass
				}
			}
			if s.PrivateKeyPath == "" {
				if envKey := os.Getenv("SSH_PRIVATE_KEY_PATH"); envKey != "" {
					s.PrivateKeyPath = envKey
				}
			}
			if s.PrivateKeyPassphrase == "" {
				if envPassphrase := os.Getenv("SSH_PRIVATE_KEY_PASSPHRASE"); envPassphrase != "" {
					s.PrivateKeyPassphrase = envPassphrase
				}
			}
			if s.KnownHostsPath == "" {
				if envKnownHosts := os.Getenv("SSH_KNOWN_HOSTS_PATH"); envKnownHosts != "" {
					s.KnownHostsPath = envKnownHosts
				}
			}
		}
	}

	// Fallback auth password jika diaktifkan
	if cfg.Auth.Enabled && cfg.Auth.Password == "" {
		if envAuthPass := os.Getenv("AUTH_PASSWORD"); envAuthPass != "" {
			cfg.Auth.Password = envAuthPass
		}
	}

	// default values
	if cfg.CheckIntervalSeconds <= 0 {
		cfg.CheckIntervalSeconds = 30
	}
	if cfg.WebPort == "" {
		cfg.WebPort = "8080"
	}
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "./data/monitoring.db"
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 7 // Default simpan data 7 hari
	}

	return &cfg, nil
}
