package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var user, privateKeyPath, tagKey, tagValue, ipType, command string

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Execute SSH command across multiple targets",
	Run:   executeSSHAcrossTargets,
}

func executeSSHAcrossTargets(cmd *cobra.Command, args []string) {
	targets, err := createTargetList(tagKey, tagValue, ipType)
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

			var outputBuffer bytes.Buffer
			err := executeSSH(&outputBuffer, id, ip)
			if err != nil {
				failedTargets[ip] = err
			}
			fmt.Print(outputBuffer.String())
		}(id, ip)
	}

	wg.Wait()

	for ip, err := range failedTargets {
		fmt.Printf("Failed for IP %s: %v\n", ip, err)
	}

	fmt.Println("finish")
}

func executeSSH(outputBuffer *bytes.Buffer, id, ip string) error {
	return sshExecuteCommand(outputBuffer, user, privateKeyPath, id, ip, command, true)
}

func displaySSHHeader(outputBuffer *bytes.Buffer, id, ip, command string) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", id))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", ip))
	outputBuffer.WriteString(fmt.Sprintf("Command: %v\n", command))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

func sshExecuteCommand(outputBuffer *bytes.Buffer, user, privateKeyPath, id, ip, command string, displayHeader bool) error {
	config, err := getSSHConfig(privateKeyPath, user)
	if err != nil {
		return fmt.Errorf("failed to get ssh config: %v", err)
	}

	client, err := establishSSHConnection(ip, config)
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
		displaySSHHeader(outputBuffer, id, ip, command)
	}

	outputBuffer.WriteString(b.String())
	outputBuffer.WriteString("\n")

	return nil
}

func init() {
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringVarP(&tagKey, "tag-key", "k", "Name", "Tag key")
	sshCmd.Flags().StringVarP(&tagValue, "tag-value", "v", "", "Tag value")
	sshCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "Username for SSH")
	sshCmd.Flags().StringVarP(&privateKeyPath, "private-key", "p", "~/.ssh/id_rsa", "Path to private key")
	sshCmd.Flags().StringVarP(&ipType, "ip-type", "t", "private", "Select IP type: public or private")
	sshCmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute via SSH")
}
