package cmd

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
	"github.com/yasuyuki0321/psh/pkg/aws"
	"github.com/yasuyuki0321/psh/pkg/scputils"
	"github.com/yasuyuki0321/psh/pkg/sshutils"
	"github.com/yasuyuki0321/psh/pkg/utils"
)

var source, dest, permission string
var decompress, createDir bool

var scpCmd = &cobra.Command{
	Use:   "scp",
	Short: "execute scp operations across multiple targets",
	Run:   runScp,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if port != 22 && (port < 1024 || port > 65535) {
			return fmt.Errorf("port value %d is out of the range 1024-65535 or not equal to 22", port)
		}
		return nil
	},
}

func runScp(cmd *cobra.Command, args []string) {

	scpConfig := scputils.ScpConfig{
		User:        user,
		PrivateKey:  privateKeyPath,
		Port:        port,
		Source:      source,
		Destination: dest,
		Permission:  permission,
		Decompress:  decompress,
		CreateDir:   createDir,
	}

	sshConfig := sshutils.SshConfig{
		User:       user,
		PrivateKey: privateKeyPath,
		Port:       port,
		Command:    command,
	}

	if tags == "" {
		if !utils.ConfirmNoTagPrompt() {
			fmt.Println("Operation aborted by user due to lack of specified tags.")
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
		if !scputils.DisplayScpPreview(targets, &scpConfig) {
			fmt.Println("Operation aborted.")
			return
		}
	}

	var mtx = sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(targets))
	failedTargets := make(map[aws.InstanceInfo]error)

	for _, target := range targets {
		go func(target aws.InstanceInfo) {
			defer wg.Done()

			var outputBuffer bytes.Buffer
			err := scputils.ExecuteScpOnTarget(&outputBuffer, &scpConfig, &sshConfig, target)
			if err != nil {
				mtx.Lock()
				failedTargets[target] = err
				mtx.Unlock()
			}
		}(target)
	}

	wg.Wait()

	for _, value := range failedTargets {
		fmt.Printf("failed to execute scp err: %v\n", value)
	}

	fmt.Println("finish")
}

func init() {
	rootCmd.AddCommand(scpCmd)

	scpCmd.Flags().StringVarP(&tags, "tags", "t", "", "comma-separated list of tag key=value pairs. Example: Key1=Value1,Key2=Value2")
	scpCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "username to execute SCP command")
	scpCmd.Flags().StringVarP(&privateKeyPath, "private-key", "k", "~/.ssh/id_rsa", "path to private key")
	scpCmd.Flags().IntVarP(&port, "port", "p", 22, "port number for SSH")
	scpCmd.Flags().StringVarP(&ipType, "ip-type", "i", "private", "select IP type: public or private")
	scpCmd.Flags().StringVarP(&source, "source", "s", "", "source file")
	scpCmd.MarkFlagRequired("source")
	scpCmd.Flags().StringVarP(&dest, "dest", "d", "", "dest file")
	scpCmd.MarkFlagRequired("dest")
	scpCmd.Flags().StringVarP(&permission, "permission", "m", "644", "permission")
	scpCmd.Flags().BoolVarP(&decompress, "decompress", "z", false, "decompress the file after SCP")
	scpCmd.Flags().BoolVarP(&createDir, "create-dir", "c", false, "create the directory if it doesn't exist")
	scpCmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the preview and execute the SCP directly")
}
