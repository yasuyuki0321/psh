package cmd

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

func getDecompressCommand(filePath string) (string, error) {
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

func isCommandAvailableOnRemote(user, privateKeyPath, commandName string, target InstanceInfo) (bool, error) {
	testCmd := fmt.Sprintf("command -v %s", commandName)
	outputBuffer := &bytes.Buffer{}

	err := sshExecuteCommand(outputBuffer, user, privateKeyPath, target, testCmd, false)
	if err != nil || strings.TrimSpace(outputBuffer.String()) == "" {
		return false, nil
	}
	return true, nil
}
