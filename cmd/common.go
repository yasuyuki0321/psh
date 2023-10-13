package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const timeOut = 5

func ParseTags(tags string) map[string]string {
	tagMap := make(map[string]string)

	pairs := strings.Split(tags, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			tagMap[parts[0]] = parts[1]
		}
	}
	return tagMap
}

func getHomePath(path string) string {
	if path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, path[2:])
}

func getSSHConfig(privateKeyPath, user string) (*ssh.ClientConfig, error) {
	keyPath := getHomePath(privateKeyPath)

	key, err := os.ReadFile(keyPath)
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

	ctx, cancel := context.WithTimeout(context.Background(), timeOut*time.Second)
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
		return nil, fmt.Errorf("ssh connection timed out after %d seconds", timeOut)
	case err := <-errorCh:
		return nil, err
	case client := <-resultCh:
		return client, nil
	}
}

func isDirectoryExistsOnRemote(user string, privateKeyPath string, target InstanceInfo, dirPath string) (bool, error) {
	checkCmd := fmt.Sprintf("[ -d %s ] && echo 'exists' || echo 'not exists'", dirPath)
	outputBuffer := &bytes.Buffer{}

	err := sshExecuteCommand(outputBuffer, user, privateKeyPath, target, checkCmd, false)
	if err != nil {
		return false, err
	}

	output := strings.TrimSpace(outputBuffer.String())
	if output == "exists" {
		return true, nil
	} else if output == "not exists" {
		return false, nil
	} else {
		return false, fmt.Errorf("unexpected output: %s", output)
	}
}
