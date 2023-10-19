package scputils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/yasuyuki0321/psh/pkg/aws"
	pshSsh "github.com/yasuyuki0321/psh/pkg/ssh"
	"github.com/yasuyuki0321/psh/pkg/sshutils"
	"github.com/yasuyuki0321/psh/pkg/utils"
)

type ScpConfig struct {
	User        string
	PrivateKey  string
	Port        int
	Source      string
	Destination string
	Permission  string
	Decompress  bool
	CreateDir   bool
}

func DisplayScpPreview(targets map[string]aws.InstanceInfo, scpConfig *ScpConfig) bool {
	fmt.Println("Targets:")
	for _, target := range targets {
		fmt.Printf("Name: %s / ID: %s / IP: %s\n", target.Name, target.ID, target.IP)
	}

	fmt.Printf("\nSource: %s\nDestination: %s\nPermission: %s\n", scpConfig.Source, scpConfig.Destination, scpConfig.Permission)
	if scpConfig.Decompress {
		fmt.Println("Decompression: Enabled")
	}
	if scpConfig.CreateDir {
		fmt.Println("Directory Creation: Enabled")
	}

	fmt.Print("\nDo you want to continue? [y/N]: ")
	var response string
	fmt.Scan(&response)

	return strings.ToLower(response) == "y"
}

func ExecuteScpOnTarget(outputBuffer *bytes.Buffer, scpConfig *ScpConfig, sshConfig *sshutils.SshConfig, target aws.InstanceInfo) error {
	err := scpExec(outputBuffer, scpConfig, sshConfig, target)
	if err != nil {
		return fmt.Errorf("error executing on %v: %v", target.IP, err)
	}

	fmt.Print(outputBuffer.String())
	return nil
}

func printScpHeader(outputBuffer *bytes.Buffer, scpConfig *ScpConfig, target aws.InstanceInfo) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("Name: %v\n", target.Name))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", target.ID))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", target.IP))
	outputBuffer.WriteString(fmt.Sprintf("Source: %v\n", scpConfig.Source))
	outputBuffer.WriteString(fmt.Sprintf("Dest: %v\n", scpConfig.Destination))
	outputBuffer.WriteString(fmt.Sprintf("Permission: %v\n", scpConfig.Permission))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

func createScpClient(target aws.InstanceInfo, scpConfig *ScpConfig) (*scp.Client, *ssh.Client, error) {
	clientConfig, err := pshSsh.GetSSHConfig(scpConfig.PrivateKey, scpConfig.User)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get ssh config: %v", err)
	}

	client, err := pshSsh.EstablishSSHConnection(target.IP, scpConfig.Port, clientConfig)
	if err != nil {
		return nil, nil, err
	}

	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		client.Close() // Don't forget to close the client if there's an error.
		return nil, nil, fmt.Errorf("error creating new SSH session from existing connection: %v", err)
	}

	return &scpClient, client, nil
}

func scpExec(outputBuffer *bytes.Buffer, scpConfig *ScpConfig, sshConfig *sshutils.SshConfig, target aws.InstanceInfo) error {
	scpClient, sshClient, err := createScpClient(target, scpConfig)
	if err != nil {
		return err
	}
	defer scpClient.Close()
	defer sshClient.Close()

	file, err := os.Open(scpConfig.Source)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	destDir := filepath.Dir(scpConfig.Destination)
	exists, err := IsDirectoryExistsOnRemote(sshConfig, target, destDir)
	if err != nil {
		return fmt.Errorf("error checking directory existence: %v", err)
	}

	if !exists {
		if scpConfig.CreateDir {
			sshConfig.Command = "mkdir -p " + destDir
			err := sshutils.SshExecuteCommand(outputBuffer, sshConfig, target, false)
			if err != nil {
				return fmt.Errorf("failed to create directory %s on %s: %v", destDir, target.IP, err)
			}
		} else {
			return fmt.Errorf("destination directory %s does not exist on %s", destDir, target.IP)
		}
	}

	err = scpClient.CopyFromFile(context.Background(), *file, scpConfig.Destination, scpConfig.Permission)
	if err != nil {
		return fmt.Errorf("error while copying file: %v", err)
	}

	printScpHeader(outputBuffer, scpConfig, target)

	if scpConfig.Decompress {
		decompressCmd, err := utils.GetDecompressCommand(scpConfig.Destination)
		if err != nil {
			return fmt.Errorf("could not get decompress command: %v", err)
		}

		cmdAvailable, err := IsCommandAvailableOnRemote(sshConfig, strings.Fields(decompressCmd)[0], target)
		if err != nil {
			return fmt.Errorf("error checking command availability: %v", err)
		}

		if cmdAvailable {
			sshConfig.Command = decompressCmd
			err = sshutils.SshExecuteCommand(outputBuffer, sshConfig, target, false)
			if err != nil {
				return fmt.Errorf("error decompressing file on %v: %v", target.IP, err)
			}
		} else {
			return fmt.Errorf("decompression command not available on remote")
		}
	}

	switch {
	case scpConfig.Decompress:
		directory := filepath.Dir(scpConfig.Destination)
		sshConfig.Command = "ls -lart " + directory
	default:
		sshConfig.Command = "ls -lart " + scpConfig.Destination
	}

	err = sshutils.SshExecuteCommand(outputBuffer, sshConfig, target, false)
	if err != nil {
		return fmt.Errorf("failed to execute ls command: %v", err)
	}

	return nil
}

// IsCommandAvailableOnRemote はリモートサーバー上で特定のコマンドが利用可能か確認する
func IsCommandAvailableOnRemote(config *sshutils.SshConfig, commandName string, target aws.InstanceInfo) (bool, error) {
	config.Command = fmt.Sprintf("command -v %s", commandName)
	outputBuffer := &bytes.Buffer{}

	err := sshutils.ExecuteSSH(outputBuffer, config, target, false)
	if err != nil || strings.TrimSpace(outputBuffer.String()) == "" {
		return false, nil
	}
	return true, nil
}

// IsDirectoryExistsOnRemote はリモートサーバー上に指定されたディレクトリが存在するか確認します。
func IsDirectoryExistsOnRemote(sshConfig *sshutils.SshConfig, target aws.InstanceInfo, dirPath string) (bool, error) {
	sshConfig.Command = fmt.Sprintf("[ -d '%s' ] && echo 'exists' || echo 'not exists'", dirPath)
	outputBuffer := &bytes.Buffer{}

	err := sshutils.ExecuteSSH(outputBuffer, sshConfig, target, false)
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
