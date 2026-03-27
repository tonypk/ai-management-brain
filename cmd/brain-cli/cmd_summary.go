package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate and send daily summary",
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !confirm("Generate and send daily summary?") {
			fmt.Println("Cancelled.")
			return
		}

		data, err := client.Post("/openclaw/summary", map[string]string{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			Summary        string `json:"summary"`
			SubmissionRate string `json:"submission_rate"`
			SentTo         string `json:"sent_to"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s Summary sent via %s (rate: %s)\n\n", green("✓"), resp.SentTo, resp.SubmissionRate)
		fmt.Println(resp.Summary)
	},
}

func init() {
	summaryCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
}
