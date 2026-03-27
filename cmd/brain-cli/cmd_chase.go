package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var chaseCmd = &cobra.Command{
	Use:   "chase [name]",
	Short: "Chase non-submitters",
	Long:  "Send chase reminders to employees who haven't submitted today, or a specific employee.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		body := map[string]string{}
		target := "non-submitters"
		if len(args) > 0 {
			body["name"] = args[0]
			target = args[0]
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !confirm(fmt.Sprintf("Chase %s?", target)) {
			fmt.Println("Cancelled.")
			return
		}

		data, err := client.Post("/openclaw/chase", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			Chased  int `json:"chased"`
			Skipped int `json:"skipped"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s Chased %d employee(s)", green("✓"), resp.Chased)
		if resp.Skipped > 0 {
			fmt.Printf(", %d skipped", resp.Skipped)
		}
		fmt.Println()
	},
}

func init() {
	chaseCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
}
