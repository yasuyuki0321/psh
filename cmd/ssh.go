package cmd

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	"github.com/yasuyuki0321/psh/pkg/aws"
	"github.com/yasuyuki0321/psh/pkg/sshutils"
	"github.com/yasuyuki0321/psh/pkg/utils"
)

var user, privateKeyPath, tags, ipType, command, argument string
var port int
var yes bool
var sshConfig sshutils.SshConfig

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "execute SSH command across multiple targets",
	Run:   runSsh,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if port != 22 && (port < 1024 || port > 65535) {
			return fmt.Errorf("port value %d is out of the range 1024-65535 or not equal to 22", port)
		}
		return nil
	},
}

func runSsh(cmd *cobra.Command, args []string) {

	sshConfig := sshutils.SshConfig{
		User:       user,
		PrivateKey: privateKeyPath,
		Port:       port,
		Command:    command,
	}

	// タグが指定されていない場合の確認処理する
	if tags == "" {
		if !utils.ConfirmNoTagPrompt() {
			fmt.Println("Operation aborted by user due to lack of specified tags.")
			return
		}
	}

	// タグの解析
	tags := utils.ParseTags(tags)

	// 対象となるインスタンスのリストの生成する
	targets, err := aws.CreateTargetList(tags, ipType)
	if err != nil {
		fmt.Printf("failed to create target list: %v\n", err)
		return
	}

	// ターゲットとコマンドのプレビュー表示する
	if !yes && !sshutils.PreviewTargets(targets, command) {
		fmt.Println("operation aborted.")
		return
	}

	// 各ターゲットにSSH接続してコマンドを実行する
	var mtx sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(targets))
	failedTargets := make(map[aws.InstanceInfo]error)

	for _, target := range targets {
		go func(target aws.InstanceInfo) {
			defer wg.Done()

			var outputBuffer bytes.Buffer
			err := sshutils.ExecuteSSH(&outputBuffer, &sshConfig, target, true)
			if err != nil {
				mtx.Lock()
				failedTargets[target] = err
				mtx.Unlock()
			}
			fmt.Print(outputBuffer.String())
		}(target)
	}
	wg.Wait()

	// 失敗したターゲットの情報表示する
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
