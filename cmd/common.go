package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

func getSSHConfig(privateKeyPath, user string) (*ssh.ClientConfig, error) {
	key, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from %s: %v", privateKeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func establishSSHConnection(ip string, config *ssh.ClientConfig) (*ssh.Client, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultCh := make(chan *ssh.Client)
	errorCh := make(chan error)

	go func() {
		client, err := ssh.Dial("tcp", ip+":22", config)
		if err != nil {
			errorCh <- err
			return
		}
		resultCh <- client
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("SSH connection timed out after 5 seconds")
	case err := <-errorCh:
		return nil, err
	case client := <-resultCh:
		return client, nil
	}
}
