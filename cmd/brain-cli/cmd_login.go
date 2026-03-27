package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Configure API key and server",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		cfg, _ := loadConfig()

		fmt.Printf("Server [%s]: ", cfg.Server)
		if server, _ := reader.ReadString('\n'); strings.TrimSpace(server) != "" {
			cfg.Server = strings.TrimSpace(server)
		}

		fmt.Print("API Key: ")
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key == "" {
			fmt.Fprintln(os.Stderr, "API key is required.")
			os.Exit(1)
		}
		cfg.APIKey = key

		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\n%s Config saved to %s\n", green("✓"), configPath())
	},
}
