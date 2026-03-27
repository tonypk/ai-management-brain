package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "brain",
	Short: "CLI for AI Management Brain",
	Long:  "Terminal-native access to manageaibrain.com — manage your team from the command line.",
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(alertsCmd)
	rootCmd.AddCommand(employeesCmd)
	rootCmd.AddCommand(risksCmd)
	rootCmd.AddCommand(kpisCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(checkinCmd)
	rootCmd.AddCommand(chaseCmd)
	rootCmd.AddCommand(summaryCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
