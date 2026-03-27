package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var checkinCmd = &cobra.Command{
	Use:   "checkin [name]",
	Short: "Send check-in to employees",
	Long:  "Send check-in questions to all employees, or a specific employee by name.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		body := map[string]string{}
		target := "all employees"
		if len(args) > 0 {
			body["name"] = args[0]
			target = args[0]
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes && !confirm(fmt.Sprintf("Send check-in to %s?", target)) {
			fmt.Println("Cancelled.")
			return
		}

		data, err := client.Post("/openclaw/checkin", body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			SentTo  int `json:"sent_to"`
			Skipped int `json:"skipped"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s Check-in sent to %d employee(s)", green("✓"), resp.SentTo)
		if resp.Skipped > 0 {
			fmt.Printf(", %d skipped", resp.Skipped)
		}
		fmt.Println()
	},
}

func init() {
	checkinCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
}

func confirm(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	var answer string
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y" || answer == "yes"
}
