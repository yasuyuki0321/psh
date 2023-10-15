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

func ExecuteSSH(outputBuffer *bytes.Buffer, port int, user string, privateKeyPath string, target aws.InstanceInfo, command string) error {
	err := SshExecuteCommand(outputBuffer, port, user, privateKeyPath, target, command, true)

	if err != nil {
		logger.LogCommandExecution(target, command, err)
	} else {
		logger.LogCommandExecution(target, command, err)
	}
	return err
}

func DisplaySSHHeader(outputBuffer *bytes.Buffer, target aws.InstanceInfo, command string) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", target.ID))
	outputBuffer.WriteString(fmt.Sprintf("Name: %v\n", target.Name))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", target.IP))
	outputBuffer.WriteString(fmt.Sprintf("Command: %v\n", command))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

func SshExecuteCommand(outputBuffer *bytes.Buffer, port int, user string, privateKeyPath string, target aws.InstanceInfo, command string, displayHeader bool) error {
	config, err := ssh.GetSSHConfig(privateKeyPath, user)
	if err != nil {
		return fmt.Errorf("failed to get ssh config: %v", err)
	}

	client, err := ssh.EstablishSSHConnection(target.IP, port, config)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(command); err != nil {
		return fmt.Errorf("failed to run command: %v", err)
	}

	if displayHeader {
		DisplaySSHHeader(outputBuffer, target, command)
	}

	outputBuffer.WriteString(b.String())
	outputBuffer.WriteString("\n")

	return nil
}

func IsCommandAvailableOnRemote(port int, user, privateKeyPath, commandName string, target aws.InstanceInfo) (bool, error) {
	testCmd := fmt.Sprintf("command -v %s", commandName)
	outputBuffer := &bytes.Buffer{}

	err := SshExecuteCommand(outputBuffer, port, user, privateKeyPath, target, testCmd, false)
	if err != nil || strings.TrimSpace(outputBuffer.String()) == "" {
		return false, nil
	}
	return true, nil
}

func IsDirectoryExistsOnRemote(port int, user string, privateKeyPath string, target aws.InstanceInfo, dirPath string) (bool, error) {
	checkCmd := fmt.Sprintf("[ -d %s ] && echo 'exists' || echo 'not exists'", dirPath)
	outputBuffer := &bytes.Buffer{}

	err := SshExecuteCommand(outputBuffer, port, user, privateKeyPath, target, checkCmd, false)
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
