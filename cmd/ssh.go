package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// Global flags
var user, privateKeyPath, tagKey, tagValue, command string

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Execute SSH command across multiple targets",
	Run:   executeSSHAcrossTargets,
}

func executeSSHAcrossTargets(cmd *cobra.Command, args []string) {
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

			err := executeSSH(id, ip)
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

func executeSSH(id string, ip string) error {
	err := sshExecuteCommand(user, privateKeyPath, id, ip, command)
	if err != nil {
		fmt.Printf("Error executing on %v: %v\n", ip, err)
		return err
	}
	return nil
}

func displaySSHHeader(id, ip, command string) {
	fmt.Println(strings.Repeat("-", 10))
	fmt.Printf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("ID: %v\n", id)
	fmt.Printf("IP: %v\n", ip)
	fmt.Printf("Command: %v\n", command)
	fmt.Println(strings.Repeat("-", 10))
}

func sshExecuteCommand(user, privateKeyPath, id, ip, command string) error {
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

	displaySSHHeader(id, ip, command)
	fmt.Println(b.String())

	return nil
}

func init() {
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "Username for SSH")
	sshCmd.Flags().StringVarP(&privateKeyPath, "private-key", "p", "~/.ssh/id_rsa", "Path to private key")
	sshCmd.Flags().StringVarP(&tagKey, "tag-key", "k", "Name", "Tag key")
	sshCmd.Flags().StringVarP(&tagValue, "tag-value", "v", "", "Tag value")
	sshCmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute via SSH")
}
