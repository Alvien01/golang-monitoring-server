package sshclient

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"monitoring-app/internal/config"
)

// Connect membuka koneksi SSH ke server berdasarkan config.
// Mendukung auth via password ATAU private key.
func Connect(cfg config.ServerConfig) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if cfg.PrivateKeyPath != "" {
		key, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("gagal baca private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("gagal parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // NOTE: untuk production sebaiknya verifikasi host key asli
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal konek ssh ke %s: %w", addr, err)
	}
	return client, nil
}

// RunCommand menjalankan satu command di server via SSH dan mengembalikan output-nya
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
