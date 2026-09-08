package sshclient

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"monitoring-app/internal/config"
)

func Connect(cfg config.ServerConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if cfg.PrivateKeyPath != "" {
		key, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("gagal baca private key: %w", err)
		}

		var signer ssh.Signer
		if cfg.PrivateKeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cfg.PrivateKeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return nil, fmt.Errorf("gagal parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	} else {
		return nil, fmt.Errorf("autentikasi gagal: password atau private_key_path harus diisi (via config.yaml atau env var SSH_PASSWORD / SSH_PRIVATE_KEY_PATH)")
	}

	hostKeyCallback, err := getHostKeyCallback(cfg)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal konek ssh ke %s: %w", addr, err)
	}
	return client, nil
}

func getHostKeyCallback(cfg config.ServerConfig) (ssh.HostKeyCallback, error) {
	if cfg.InsecureIgnoreHostKey {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	knownHostsPath := cfg.KnownHostsPath
	if knownHostsPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, ".ssh", "known_hosts")
			if _, statErr := os.Stat(defaultPath); statErr == nil {
				knownHostsPath = defaultPath
			}
		}
	}

	if knownHostsPath == "" {
		return nil, fmt.Errorf("verifikasi host key gagal: file known_hosts tidak ditemukan. Buat file known_hosts, tentukan 'known_hosts_path', atau set 'insecure_ignore_host_key: true' di config.yaml")
	}

	cb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat known_hosts dari '%s': %w", knownHostsPath, err)
	}

	return cb, nil
}

func RunCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("gagal buka session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("gagal jalankan command '%s': %w", cmd, err)
	}
	return string(output), nil
}
