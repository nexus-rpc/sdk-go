package main

import (
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "sdk-go",
		Short: "Scripts for the Nexus Go SDK repo",
	}

	rootCmd.AddCommand(installDepsCmd())
	rootCmd.AddCommand(buildCmd())
	rootCmd.AddCommand(cleanCmd())

	if err := rootCmd.Execute(); err != nil {
		BackupLogger.Fatalf("Failed to run %s", err)
	}
}
