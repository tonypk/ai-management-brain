package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report [weekly|monthly]",
	Short: "Show team performance report",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		period := "weekly"
		if len(args) > 0 {
			period = args[0]
		}

		data, err := client.Get("/openclaw/report?period=" + period)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			Period         string `json:"period"`
			DateRange      struct {
				Start string `json:"start"`
				End   string `json:"end"`
			} `json:"date_range"`
			SubmissionRate string `json:"submission_rate"`
			Ranking        []struct {
				Name  string `json:"name"`
				Days  int    `json:"days"`
				Medal string `json:"medal"`
			} `json:"ranking"`
			OneOnOneSuggestions []struct {
				Name string `json:"name"`
				Days int    `json:"days"`
			} `json:"one_on_one_suggestions"`
		}

		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s Report — %s to %s\n", bold(capitalize(resp.Period)),
			resp.DateRange.Start, resp.DateRange.End)
		fmt.Printf("Submission rate: %s\n\n", bold(resp.SubmissionRate))

		if len(resp.Ranking) > 0 {
			fmt.Println(bold("Rankings:"))
			rows := make([][]string, len(resp.Ranking))
			for i, r := range resp.Ranking {
				medal := medalIcon(r.Medal)
				rows[i] = []string{medal, r.Name, fmt.Sprintf("%d days", r.Days)}
			}
			table([]string{"", "Name", "Days"}, rows)
		}

		if len(resp.OneOnOneSuggestions) > 0 {
			fmt.Printf("\n%s\n", bold("1:1 Suggestions:"))
			for _, s := range resp.OneOnOneSuggestions {
				fmt.Printf("  %s %s (%d days missed)\n", yellow("→"), s.Name, s.Days)
			}
		}
	},
}

func medalIcon(medal string) string {
	switch medal {
	case "gold":
		return "🥇"
	case "silver":
		return "🥈"
	case "bronze":
		return "🥉"
	default:
		return "  "
	}
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
