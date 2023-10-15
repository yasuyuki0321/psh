package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ParseTags(tags string) map[string]string {
	tagMap := make(map[string]string)

	pairs := strings.Split(tags, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			tagMap[parts[0]] = parts[1]
		}
	}
	return tagMap
}

func GetHomePath(path string) string {
	if path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, path[2:])
}

func GetDecompressCommand(filePath string) (string, error) {
	directory := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)

	switch {
	case strings.HasSuffix(filePath, ".tar.gz"):
		return fmt.Sprintf("cd %s && tar -xzf %s", directory, fileName), nil
	case strings.HasSuffix(filePath, ".tar"):
		return fmt.Sprintf("cd %s && tar -xf %s", directory, fileName), nil
	case strings.HasSuffix(filePath, ".gz"):
		return fmt.Sprintf("cd %s && gunzip -df %s", directory, fileName), nil
	case strings.HasSuffix(filePath, ".zip"):
		return fmt.Sprintf("cd %s && unzip %s", directory, fileName), nil
	default:
		return "", fmt.Errorf("unsupported file extension for %v", filePath)
	}
}

// func IsCommandAvailableOnRemote(port, user, privateKeyPath, commandName string, target aws.InstanceInfo) (bool, error) {
// 	testCmd := fmt.Sprintf("command -v %s", commandName)
// 	outputBuffer := &bytes.Buffer{}

// 	err := sshutils.ExecuteCommand(outputBuffer, port, user, privateKeyPath, target, testCmd, false)
// 	if err != nil || strings.TrimSpace(outputBuffer.String()) == "" {
// 		return false, nil
// 	}
// 	return true, nil
// }

// func IsDirectoryExistsOnRemote(port int, user string, privateKeyPath string, target aws.InstanceInfo, dirPath string) (bool, error) {
// 	checkCmd := fmt.Sprintf("[ -d %s ] && echo 'exists' || echo 'not exists'", dirPath)
// 	outputBuffer := &bytes.Buffer{}

// 	err := sshutils.SshExecuteCommand(outputBuffer, port, user, privateKeyPath, target, checkCmd, false)
// 	if err != nil {
// 		return false, err
// 	}

// 	output := strings.TrimSpace(outputBuffer.String())
// 	if output == "exists" {
// 		return true, nil
// 	} else if output == "not exists" {
// 		return false, nil
// 	} else {
// 		return false, fmt.Errorf("unexpected output: %s", output)
// 	}
// }