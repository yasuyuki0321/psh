package cmd

import (
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

// scpCmd represents the scp command
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

	for id, ip := range targets {
		go executeScpOnTarget(&wg, id, ip)
	}

	wg.Wait()
	fmt.Println("finish")
}

func executeScpOnTarget(wg *sync.WaitGroup, id string, ip string) {
	defer wg.Done()

	err := scpExec(user, privateKeyPath, id, ip, source, dest, permission)
	if err != nil {
		fmt.Printf("error executing on %v: %v\n", ip, err)
	}
}

func printScpHeader(id, ip, source, dest, permission string) {
	fmt.Println(strings.Repeat("-", 10))
	fmt.Printf("time: %v\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("id: %v\n", id)
	fmt.Printf("ip: %v\n", ip)
	fmt.Printf("source: %v\n", source)
	fmt.Printf("dest: %v\n", dest)
	fmt.Printf("permission: %v\n", permission)
	fmt.Println(strings.Repeat("-", 10))
}

func scpExec(user, privateKeyPath, id, ip, source, dest, permission string) error {
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

	printScpHeader(id, ip, source, dest, permission)
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
