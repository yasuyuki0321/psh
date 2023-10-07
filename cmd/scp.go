package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/spf13/cobra"
)

var source, dest, permission string

var scpCmd = &cobra.Command{
	Use:   "scp",
	Short: "A command to perform scp operations across multiple targets",
	Run:   runScp,
}

func runScp(cmd *cobra.Command, args []string) {
	targets, err := createTargetList(tagKey, tagValue)
	if err != nil {
		fmt.Printf("Failed to create target list: %v\n", err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(targets))

	failedTargets := make(map[string]error)

	for id, ip := range targets {
		go func(id, ip string) {
			defer wg.Done()

			err := executeScpOnTarget(id, ip)
			if err != nil {
				failedTargets[ip] = err
			}
		}(id, ip)
	}

	wg.Wait()

	for ip, err := range failedTargets {
		fmt.Printf("Failed for IP %s: %v\n", ip, err)
	}

	fmt.Println("finish")
}

func executeScpOnTarget(id string, ip string) error {
	var outputBuffer bytes.Buffer

	err := scpExec(&outputBuffer, user, privateKeyPath, id, ip, source, dest, permission)
	if err != nil {
		outputBuffer.WriteString(fmt.Sprintf("error executing on %v: %v\n", ip, err))
	}
	fmt.Print(outputBuffer.String())
	return err
}

func printScpHeader(outputBuffer *bytes.Buffer, id, ip, source, dest, permission string) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", id))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", ip))
	outputBuffer.WriteString(fmt.Sprintf("Source: %v\n", source))
	outputBuffer.WriteString(fmt.Sprintf("Destination: %v\n", dest))
	outputBuffer.WriteString(fmt.Sprintf("Permission: %v\n", permission))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

func scpExec(outputBuffer *bytes.Buffer, user, privateKeyPath, id, ip, source, dest, permission string) error {
	config, err := getSSHConfig(privateKeyPath, user)
	if err != nil {
		return fmt.Errorf("failed to get ssh config: %v", err)
	}

	client, err := establishSSHConnection(ip, config)
	if err != nil {
		return err
	}
	defer client.Close()

	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		return fmt.Errorf("error creating new SSH session from existing connection: %v", err)
	}
	defer scpClient.Close()

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	err = scpClient.CopyFromFile(context.Background(), *file, dest, permission)
	if err != nil {
		return fmt.Errorf("error while copying file: %v", err)
	}

	printScpHeader(outputBuffer, id, ip, source, dest, permission)

	cmd := "ls -l " + dest
	err = sshExecuteCommand(outputBuffer, user, privateKeyPath, id, ip, cmd, false)
	if err != nil {
		return fmt.Errorf("failed to execute ls command: %v", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(scpCmd)

	scpCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "username to execute ssh command")
	scpCmd.Flags().StringVarP(&privateKeyPath, "private-key", "p", "~/.ssh/id_rsa", "path to private key")
	scpCmd.Flags().StringVarP(&tagKey, "tag-key", "k", "Name", "tag key")
	scpCmd.Flags().StringVarP(&tagValue, "tag-value", "v", "", "tag value")
	scpCmd.Flags().StringVarP(&source, "source", "s", "", "source file")
	scpCmd.Flags().StringVarP(&dest, "dest", "d", "", "dest file")
	scpCmd.Flags().StringVarP(&permission, "permission", "m", "", "permission")
}
