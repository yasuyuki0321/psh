package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/yasuyuki0321/psh/pkg/aws"
	"github.com/yasuyuki0321/psh/pkg/sshutils"
	"github.com/yasuyuki0321/psh/pkg/utils"
)

var user, privateKeyPath, tags, ipType, command string
var port int
var yes bool

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "execute SSH command across multiple targets",
	Run:   executeSSHAcrossTargets,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if port != 22 && (port < 1024 || port > 65535) {
			return fmt.Errorf("port value %d is out of the range 1024-65535 or not equal to 22", port)
		}
		return nil
	},
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

	tags := utils.ParseTags(tags)
	targets, err := aws.CreateTargetList(tags, ipType)
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
	failedTargets := make(map[aws.InstanceInfo]error)

	for _, target := range targets {
		go func(target aws.InstanceInfo) {
			defer wg.Done()

			var outputBuffer bytes.Buffer
			err := sshutils.ExecuteSSH(&outputBuffer, port, user, privateKeyPath, target, command)
			if err != nil {
				mtx.Lock()
				failedTargets[target] = err
				mtx.Unlock()
			}
			fmt.Print(outputBuffer.String())
		}(target)
	}

	wg.Wait()

	for target, value := range failedTargets {
		fmt.Printf("Failed to execute SSH command on Target [Name: %s (IP: %s)]. Error: %v\n", target.Name, target.IP, value)
	}

	fmt.Println("finish")
}

func init() {
	rootCmd.AddCommand(sshCmd)

	sshCmd.Flags().StringVarP(&tags, "tags", "t", "", "comma-separated list of tag key=value pairs Example: Key1=Value1,Key2=Value2")
	sshCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "username for SSH")
	sshCmd.Flags().StringVarP(&privateKeyPath, "private-key", "k", "~/.ssh/id_rsa", "path to private key")
	sshCmd.Flags().IntVarP(&port, "port", "p", 22, "port number for SSH")
	sshCmd.Flags().StringVarP(&ipType, "ip-type", "i", "private", "select IP type: public or private")
	sshCmd.Flags().StringVarP(&command, "command", "c", "", "command to execute via SSH")
	sshCmd.MarkFlagRequired("command")
	sshCmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the preview and execute the command directly")
}
