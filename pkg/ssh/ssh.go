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

// GetSSHConfig はSSH接続のための設定を取得する
func GetSSHConfig(privateKeyPath, user string) (*ssh.ClientConfig, error) {
	keyPath := utils.GetHomePath(privateKeyPath)

	// 秘密鍵を読み取る
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from %v: %v", keyPath, err)
	}

	// 秘密鍵を解析する
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	// SSH接続設定を返す
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

// EstablishSSHConnection はSSH接続を確立します。
func EstablishSSHConnection(ip string, port int, config *ssh.ClientConfig) (*ssh.Client, error) {

	// 接続のタイムアウトを設定
	ctx, cancel := context.WithTimeout(context.Background(), timeOut*time.Second)
	defer cancel()

	resultCh := make(chan *ssh.Client)
	errorCh := make(chan error)

	// goroutineでSSH接続を実行する
	go func() {
		client, err := ssh.Dial("tcp", ip+":"+strconv.Itoa(port), config)
		if err != nil {
			errorCh <- err
			return
		}
		resultCh <- client
	}()

	// タイムアウト、エラー、または成功した接続のいずれかを待つ
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("ssh connection timed out after %d seconds", timeOut)
	case err := <-errorCh:
		return nil, err
	case client := <-resultCh:
		return client, nil
	}
}
