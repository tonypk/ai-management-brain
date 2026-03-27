package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version and server info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("brain %s\n", bold(version))
		cfg, _ := loadConfig()
		fmt.Printf("server: %s\n", cfg.Server)
	},
}
