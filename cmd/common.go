package cmd

import (
	"fmt"
	"os"

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
	return ssh.Dial("tcp", ip+":22", config)
}
