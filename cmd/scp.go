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
}

func displayScpPreview(targets map[string]InstanceInfo) bool {
	fmt.Println("Targets:")
	for target, value := range targets {
		fmt.Printf("Name: %s / ID: %s / IP: %s\n", value.Name, target, value.IP)
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

	tags := ParseTags(tags)
	targets, err := createTargetList(tags, ipType)
	if err != nil {
		fmt.Printf("Failed to create target list: %v\n", err)
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

	failedTargets := make(map[string]error)

	for target, value := range targets {
		go func(id, ip string) {
			defer wg.Done()

			err := executeScpOnTarget(target, value.IP)
			if err != nil {
				mtx.Lock()
				failedTargets[value.IP] = err
				mtx.Unlock()
			}
		}(target, value.IP)
	}

	wg.Wait()

	for _, err := range failedTargets {
		fmt.Printf("%v\n", err)
	}

	fmt.Println("finish")
}

func executeScpOnTarget(id, ip string) error {
	var outputBuffer bytes.Buffer

	err := scpExec(&outputBuffer, user, privateKeyPath, id, ip, source, dest, permission)
	if err != nil {
		return fmt.Errorf("error executing on %v: %v", ip, err)
	}

	fmt.Print(outputBuffer.String())
	return nil
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
	var lsCmd string

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

	destDir := filepath.Dir(dest)
	exists, err := isDirectoryExistsOnRemote(user, privateKeyPath, ip, destDir)
	if err != nil {
		return fmt.Errorf("error checking directory existence: %v", err)
	}
	if !exists {
		if createDir {
			mkdirCmd := fmt.Sprintf("mkdir -p %s", destDir)
			err := sshExecuteCommand(outputBuffer, user, privateKeyPath, "", ip, mkdirCmd, false)
			if err != nil {
				return fmt.Errorf("failed to create directory %s on %s: %v", destDir, ip, err)
			}
		} else {
			return fmt.Errorf("destination directory %s does not exist on %s", destDir, ip)
		}
	}

	err = scpClient.CopyFromFile(context.Background(), *file, dest, permission)
	if err != nil {
		return fmt.Errorf("error while copying file: %v", err)
	}

	printScpHeader(outputBuffer, id, ip, source, dest, permission)

	if decompress {
		decompressCmd, err := getDecompressCommand(dest)
		if err != nil {
			return fmt.Errorf("could not get decompress command: %v", err)
		}

		cmdAvailable, err := isCommandAvailableOnRemote(user, privateKeyPath, ip, strings.Fields(decompressCmd)[0])
		if err != nil {
			return fmt.Errorf("error checking command availability: %v", err)
		}

		if cmdAvailable {
			err = sshExecuteCommand(outputBuffer, user, privateKeyPath, id, ip, decompressCmd, false)
			if err != nil {
				return fmt.Errorf("error decompressing file on %v: %v", ip, err)
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

	err = sshExecuteCommand(outputBuffer, user, privateKeyPath, id, ip, lsCmd, false)
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
	scpCmd.Flags().StringVarP(&ipType, "ip-type", "i", "private", "select IP type: public or private")
	scpCmd.Flags().StringVarP(&source, "source", "s", "", "source file")
	scpCmd.Flags().StringVarP(&dest, "dest", "d", "", "dest file")
	scpCmd.Flags().StringVarP(&permission, "permission", "m", "", "permission")
	scpCmd.Flags().BoolVarP(&decompress, "decompress", "z", false, "decompress the file after SCP")
	scpCmd.Flags().BoolVarP(&createDir, "create-dir", "c", false, "create the directory if it doesn't exist")
	scpCmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the preview and execute the SCP directly")
}
