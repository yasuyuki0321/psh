package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "psh",
	Short: "parallel shell execution",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("cmd.Execute: %v", err)
	}
}
