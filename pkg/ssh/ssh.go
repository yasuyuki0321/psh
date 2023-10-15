package ssh

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/yasuyuki0321/psh/pkg/utils"
	"golang.org/x/crypto/ssh"
)

const timeOut = 5

func GetSSHConfig(privateKeyPath, user string) (*ssh.ClientConfig, error) {
	keyPath := utils.GetHomePath(privateKeyPath)

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

func EstablishSSHConnection(ip string, port int, config *ssh.ClientConfig) (*ssh.Client, error) {

	ctx, cancel := context.WithTimeout(context.Background(), timeOut*time.Second)
	defer cancel()

	resultCh := make(chan *ssh.Client)
	errorCh := make(chan error)

	go func() {
		client, err := ssh.Dial("tcp", ip+":"+strconv.Itoa(port), config)
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
