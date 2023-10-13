package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

var user, privateKeyPath, tags, ipType, command string
var yes bool

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "execute SSH command across multiple targets",
	Run:   executeSSHAcrossTargets,
}

func executeSSHAcrossTargets(cmd *cobra.Command, args []string) {
	var mtx sync.Mutex

	if tags == "" {
		fmt.Print("You have not specified any tags. This will execute the command on ALL EC2 instances. Continue? [y/N]: ")
		var response string
		fmt.Scan(&response)
		fmt.Println()

		if strings.ToLower(response) != "y" {
			fmt.Println("operation aborted.")
			return
		}
	}

	tags := ParseTags(tags)
	targets, err := createTargetList(tags, ipType)
	if err != nil {
		fmt.Printf("failed to create target list: %v\n", err)
		return
	}

	if !yes {
		fmt.Println("Targets:")
		for target, value := range targets {
			fmt.Printf("Name: %s / ID: %s / IP: %s\n", value.Name, target, value.IP)
		}
		fmt.Printf("\nCommand: %s\n", command)

		fmt.Print("\nDo you want to continue? [y/N]: ")
		var response string
		fmt.Scan(&response)

		if strings.ToLower(response) != "y" {
			fmt.Println("operation aborted.")
			return
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(targets))
	failedTargets := make(map[InstanceInfo]error)

	for _, target := range targets {
		go func(target, value InstanceInfo) {
			defer wg.Done()

			var outputBuffer bytes.Buffer
			err := executeSSH(&outputBuffer, target)
			if err != nil {
				mtx.Lock()
				failedTargets[value] = err
				mtx.Unlock()
			}
			fmt.Print(outputBuffer.String())
		}(target, target)
	}

	wg.Wait()

	for target, value := range failedTargets {
		fmt.Printf("failed for Name: %s / IP: %s: err: %v\n", target.Name, target.IP, value)
	}

	fmt.Println("finish")
}

func executeSSH(outputBuffer *bytes.Buffer, target InstanceInfo) error {
	return sshExecuteCommand(outputBuffer, user, privateKeyPath, target, command, true)
}

func displaySSHHeader(outputBuffer *bytes.Buffer, target InstanceInfo, command string) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", target.ID))
	outputBuffer.WriteString(fmt.Sprintf("Name: %v\n", target.Name))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", target.IP))
	outputBuffer.WriteString(fmt.Sprintf("Command: %v\n", command))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

func sshExecuteCommand(outputBuffer *bytes.Buffer, user string, privateKeyPath string, target InstanceInfo, command string, displayHeader bool) error {
	config, err := getSSHConfig(privateKeyPath, user)
	if err != nil {
		return fmt.Errorf("failed to get ssh config: %v", err)
	}

	client, err := establishSSHConnection(target.IP, config)
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
		displaySSHHeader(outputBuffer, target, command)
	}

	outputBuffer.WriteString(b.String())
	outputBuffer.WriteString("\n")

	return nil
}

func init() {
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringVarP(&tags, "tags", "t", "", "comma-separated list of tag key=value pairs Example: Key1=Value1,Key2=Value2")
	sshCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "username for SSH")
	sshCmd.Flags().StringVarP(&privateKeyPath, "private-key", "k", "~/.ssh/id_rsa", "path to private key")
	sshCmd.Flags().StringVarP(&ipType, "ip-type", "i", "private", "select IP type: public or private")
	sshCmd.Flags().StringVarP(&command, "command", "c", "", "command to execute via SSH")
	sshCmd.MarkFlagRequired("command")
	sshCmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the preview and execute the command directly")
}
