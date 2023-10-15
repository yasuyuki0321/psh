package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/spf13/cobra"
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

func displayScpPreview(targets map[string]InstanceInfo) bool {
	fmt.Println("Targets:")
	for _, target := range targets {
		fmt.Printf("Name: %s / ID: %s / IP: %s\n", target.Name, target.ID, target.IP)
	}

	fmt.Printf("\nSource: %s\nDestination: %s\nPermission: %s\n", source, dest, permission)
	if decompress {
		fmt.Println("Decompression: Enabled")
	}
	if createDir {
		fmt.Println("Directory Creation: Enabled")
	}

	fmt.Print("\nDo you want to continue? [y/N]: ")
	var response string
	fmt.Scan(&response)

	return strings.ToLower(response) == "y"
}

func runScp(cmd *cobra.Command, args []string) {
	var mtx sync.Mutex

	if tags == "" {
		fmt.Print("You have not specified any tags. This will execute scp command to ALL EC2 instances. Continue? [y/N]: ")
		var response string
		fmt.Scan(&response)
		fmt.Println()

		if strings.ToLower(response) != "y" {
			fmt.Println("Operation aborted.")
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
		if !displayScpPreview(targets) {
			fmt.Println("Operation aborted.")
			return
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(targets))

	failedTargets := make(map[InstanceInfo]error)

	for _, target := range targets {
		go func(target InstanceInfo) {
			defer wg.Done()

			err := executeScpOnTarget(target)
			if err != nil {
				mtx.Lock()
				failedTargets[target] = err
				mtx.Unlock()
			}
		}(target)
	}

	wg.Wait()

	for target, value := range failedTargets {
		fmt.Printf("failed for Name: %s / IP: %s: err: %v\n", target.Name, target.IP, value)
	}

	fmt.Println("finish")
}

func executeScpOnTarget(target InstanceInfo) error {
	var outputBuffer bytes.Buffer

	err := scpExec(&outputBuffer, user, privateKeyPath, source, dest, permission, target)
	if err != nil {
		return fmt.Errorf("error executing on %v: %v", target.IP, err)
	}

	fmt.Print(outputBuffer.String())
	return nil
}

func printScpHeader(outputBuffer *bytes.Buffer, source, dest, permission string, target InstanceInfo) {
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
	outputBuffer.WriteString(fmt.Sprintf("Time: %v\n", time.Now().Format("2006-01-02 15:04:05")))
	outputBuffer.WriteString(fmt.Sprintf("ID: %v\n", target.ID))
	outputBuffer.WriteString(fmt.Sprintf("Name: %v\n", target.Name))
	outputBuffer.WriteString(fmt.Sprintf("IP: %v\n", target.IP))
	outputBuffer.WriteString(fmt.Sprintf("Source: %v\n", source))
	outputBuffer.WriteString(fmt.Sprintf("Destination: %v\n", dest))
	outputBuffer.WriteString(fmt.Sprintf("Permission: %v\n", permission))
	outputBuffer.WriteString(fmt.Sprintln(strings.Repeat("-", 10)))
}

func scpExec(outputBuffer *bytes.Buffer, user, privateKeyPath, source, dest, permission string, target InstanceInfo) error {
	var lsCmd string

	config, err := getSSHConfig(privateKeyPath, user)
	if err != nil {
		return fmt.Errorf("failed to get ssh config: %v", err)
	}

	client, err := establishSSHConnection(target.IP, port, config)
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

	destDir := filepath.Dir(dest)
	exists, err := isDirectoryExistsOnRemote(user, privateKeyPath, target, destDir)
	if err != nil {
		return fmt.Errorf("error checking directory existence: %v", err)
	}
	if !exists {
		if createDir {
			mkdirCmd := fmt.Sprintf("mkdir -p %s", destDir)
			err := sshExecuteCommand(outputBuffer, user, privateKeyPath, target, mkdirCmd, false)
			if err != nil {
				return fmt.Errorf("failed to create directory %s on %s: %v", destDir, target.IP, err)
			}
		} else {
			return fmt.Errorf("destination directory %s does not exist on %s", destDir, target.IP)
		}
	}

	err = scpClient.CopyFromFile(context.Background(), *file, dest, permission)
	if err != nil {
		return fmt.Errorf("error while copying file: %v", err)
	}

	printScpHeader(outputBuffer, source, dest, permission, target)

	if decompress {
		decompressCmd, err := getDecompressCommand(dest)
		if err != nil {
			return fmt.Errorf("could not get decompress command: %v", err)
		}

		cmdAvailable, err := isCommandAvailableOnRemote(user, privateKeyPath, strings.Fields(decompressCmd)[0], target)
		if err != nil {
			return fmt.Errorf("error checking command availability: %v", err)
		}

		if cmdAvailable {
			err = sshExecuteCommand(outputBuffer, user, privateKeyPath, target, decompressCmd, false)
			if err != nil {
				return fmt.Errorf("error decompressing file on %v: %v", target.IP, err)
			}
		} else {
			return fmt.Errorf("decompression command not available on remote")
		}
	}

	switch {
	case decompress:
		directory := filepath.Dir(dest)
		lsCmd = "ls -lart " + directory
	default:
		lsCmd = "ls -ltr " + dest
	}

	err = sshExecuteCommand(outputBuffer, user, privateKeyPath, target, lsCmd, false)
	if err != nil {
		return fmt.Errorf("failed to execute ls command: %v", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(scpCmd)

	scpCmd.Flags().StringVarP(&tags, "tags", "t", "", "comma-separated list of tag key=value pairs. Example: Key1=Value1,Key2=Value2")
	scpCmd.Flags().StringVarP(&user, "user", "u", "ec2-user", "username to execute SCP command")
	scpCmd.Flags().StringVarP(&privateKeyPath, "private-key", "k", "~/.ssh/id_rsa", "path to private key")
	scpCmd.Flags().IntVarP(&port, "port", "p", 22, "port number for SSH")
	scpCmd.Flags().StringVarP(&ipType, "ip-type", "i", "private", "select IP type: public or private")
	scpCmd.Flags().StringVarP(&source, "source", "s", "", "source file")
	sshCmd.MarkFlagRequired("source")
	scpCmd.Flags().StringVarP(&dest, "dest", "d", "", "dest file")
	sshCmd.MarkFlagRequired("dest")
	scpCmd.Flags().StringVarP(&permission, "permission", "m", "644", "permission")
	scpCmd.Flags().BoolVarP(&decompress, "decompress", "z", false, "decompress the file after SCP")
	scpCmd.Flags().BoolVarP(&createDir, "create-dir", "c", false, "create the directory if it doesn't exist")
	scpCmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the preview and execute the SCP directly")
}
