package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var user string
var privateKeyPath string
var tagKey string
var tagValue string
var command string

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "Execute SSH command across multiple targets",
	Run:   runSSH,
}

func runSSH(cmd *cobra.Command, args []string) {
	targets, err := createTargetList(tagKey, tagValue)
	if err != nil {
		fmt.Printf("Failed to create target list: %v\n", err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(len(targets))

	for _, target := range targets {
		go executeSSHOnTarget(&wg, target)
	}

	wg.Wait()
	fmt.Println("finish")
}

func executeSSHOnTarget(wg *sync.WaitGroup, target string) {
	defer wg.Done()

	err := sshExec(user, privateKeyPath, target, command)
	if err != nil {
		fmt.Printf("Error executing on %v: %v\n", target, err)
	}
}

func printSshHeader(ip, command string) {
	fmt.Println(strings.Repeat("-", 10))
	fmt.Printf("time: %v\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("ip: %v\n", ip)
	fmt.Printf("command: %v\n", command)
	fmt.Println(strings.Repeat("-", 10))
}

func sshExec(user, privateKeyPath, ip, command string) error {
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

	printSshHeader(ip, command)
	fmt.Println(b.String())

	return nil
}

func init() {
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "username to execute ssh command")
	sshCmd.Flags().StringVarP(&privateKeyPath, "private-key", "p", "~/.ssh/id_rsa", "path to private key")
	sshCmd.Flags().StringVarP(&tagKey, "tag-key", "k", "Name", "tag key")
	sshCmd.Flags().StringVarP(&tagValue, "tag-value", "v", "", "tag value")
	sshCmd.Flags().StringVarP(&command, "command", "c", "", "command to execute by ssh")
}
