package sshutils

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/yasuyuki0321/psh/pkg/aws"
	"github.com/yasuyuki0321/psh/pkg/logger"
	"github.com/yasuyuki0321/psh/pkg/ssh"
)

// SshConfig はSSH接続の設定を保持します。
type SshConfig struct {
	User       string
	PrivateKey string
	Port       int
	Command    string
	Arguments  []string
}

// PreviewTargets は、対象となるインスタンスと実行するコマンドを表示する
func PreviewTargets(targets map[string]aws.InstanceInfo, command string) bool {
	fmt.Println("Targets:")
	for target, value := range targets {
		fmt.Printf("Name: %s / ID: %s / IP: %s\n", value.Name, target, value.IP)
	}
	fmt.Printf("\nCommand: %s\n", command)

	fmt.Print("\nDo you want to continue? [y/N]: ")
	var response string
	fmt.Scan(&response)

	return strings.ToLower(response) == "y"
}

// DisplaySSHHeader はSSHの結果のヘッダー情報を出力する
func DisplaySSHHeader(outputBuffer *bytes.Buffer, sshConfig *SshConfig, target aws.InstanceInfo) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("Name: %v\n", target.Name))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", target.ID))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", target.IP))
	outputBuffer.WriteString(fmt.Sprintf("Command: %v\n", sshConfig.Command))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

// ExecuteSSH は指定したコマンドをSSHを通じて実行する
func ExecuteSSH(outputBuffer *bytes.Buffer, sshConfig *SshConfig, target aws.InstanceInfo, displayHeader bool) error {
	err := SshExecuteCommand(outputBuffer, sshConfig, target, displayHeader)

	if err != nil {
		logger.LogCommandExecution(target, sshConfig.Command, err)
	} else {
		logger.LogCommandExecution(target, sshConfig.Command, err)
	}
	return err
}

// SshExecuteCommand はSSHでコマンドを実行し、その結果を取得する
func SshExecuteCommand(outputBuffer *bytes.Buffer, config *SshConfig, target aws.InstanceInfo, displayHeader bool) error {
	clientConfig, err := ssh.GetSSHConfig(config.PrivateKey, config.User)
	if err != nil {
		return fmt.Errorf("failed to get ssh config: %v", err)
	}

	// SSH接続の確立
	client, err := ssh.EstablishSSHConnection(target.IP, config.Port, clientConfig)
	if err != nil {
		return err
	}
	defer client.Close()

	// セッションの作成
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(config.Command); err != nil {
		return fmt.Errorf("failed to run command: %v", err)
	}

	if displayHeader {
		DisplaySSHHeader(outputBuffer, config, target)
	}

	outputBuffer.WriteString(b.String())
	outputBuffer.WriteString("\n")

	return nil
}
