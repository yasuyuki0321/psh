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
	if len(path) < 2 || path[:2] != "~/" {
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

func ConfirmNoTagPrompt() bool {
	fmt.Print("You have not specified any tags. This will execute the command on ALL EC2 instances. Do you want to continue? [y/N]: ")

	var response string
	_, err := fmt.Scan(&response)
	if err != nil {
		fmt.Println("Input error: ", err)
		return false
	}
	fmt.Println()

	return strings.ToLower(response) == "y"
}
