package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var risksCmd = &cobra.Command{
	Use:   "risks",
	Short: "Show top execution risks",
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		data, err := client.Get("/openclaw/state/risks")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var risks []struct {
			Score    float64 `json:"score"`
			Type     string  `json:"type"`
			Employee string  `json:"employee"`
			Evidence string  `json:"evidence"`
		}

		if err := json.Unmarshal(data, &risks); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n\n", bold("Top Risks"))

		if len(risks) == 0 {
			fmt.Println(green("No active risks detected."))
			return
		}

		rows := make([][]string, len(risks))
		for i, r := range risks {
			severity := "ok"
			if r.Score >= 0.8 {
				severity = "critical"
			} else if r.Score >= 0.6 {
				severity = "warning"
			}
			rows[i] = []string{
				statusIcon(severity),
				fmt.Sprintf("%.2f", r.Score),
				r.Type,
				r.Employee,
				r.Evidence,
			}
		}
		table([]string{"", "Score", "Type", "Employee", "Evidence"}, rows)
	},
}
