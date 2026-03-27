package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show today's team check-in status",
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		data, err := client.Get("/openclaw/status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			Date           string `json:"date"`
			TotalEmployees int    `json:"total_employees"`
			Submitted      []struct {
				Name        string `json:"name"`
				SubmittedAt string `json:"submitted_at"`
			} `json:"submitted"`
			Pending []struct {
				Name       string `json:"name"`
				ChaseCount int    `json:"chase_count"`
			} `json:"pending"`
			ChaseCount int    `json:"chase_count"`
			Mentor     string `json:"mentor"`
			MentorName string `json:"mentor_name"`
		}

		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s — %s\n\n", bold("Team Status"), resp.Date)
		fmt.Printf("Submitted  %s/%d (%d%%)\n",
			green(fmt.Sprintf("%d", len(resp.Submitted))),
			resp.TotalEmployees,
			percent(len(resp.Submitted), resp.TotalEmployees))

		if len(resp.Pending) > 0 {
			names := make([]string, len(resp.Pending))
			for i, p := range resp.Pending {
				names[i] = p.Name
			}
			fmt.Printf("Pending    %s\n", yellow(joinNames(names)))
		}

		fmt.Printf("Chased     %d\n", resp.ChaseCount)

		if resp.MentorName != "" {
			fmt.Printf("Mentor     %s\n", dim(resp.MentorName))
		}

		if len(resp.Submitted) > 0 {
			fmt.Printf("\n%s\n", bold("Recent check-ins:"))
			for _, s := range resp.Submitted {
				fmt.Printf("  %-14s %s\n", s.Name, dim(s.SubmittedAt))
			}
		}
	},
}

func percent(a, b int) int {
	if b == 0 {
		return 0
	}
	return a * 100 / b
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		result += ", " + names[i]
	}
	return result
}
