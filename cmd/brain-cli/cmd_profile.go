package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile <name>",
	Short: "Show detailed employee profile",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		name := strings.Join(args, " ")
		data, err := client.Get("/employees/profile/" + name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			Name           string `json:"name"`
			Role           string `json:"role"`
			SubmissionRate string `json:"submission_rate"`
			Streak         int    `json:"streak"`
			SentimentTrend []struct {
				Date      string `json:"date"`
				Sentiment string `json:"sentiment"`
			} `json:"sentiment_trend"`
			RecentReports []struct {
				Date    string `json:"date"`
				Summary string `json:"summary"`
			} `json:"recent_reports"`
		}

		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s — %s\n", bold(resp.Name), dim(resp.Role))
		fmt.Printf("Submission rate: %s\n", resp.SubmissionRate)
		if resp.Streak > 0 {
			fmt.Printf("Current streak: %s days\n", green(fmt.Sprintf("%d", resp.Streak)))
		}

		if len(resp.SentimentTrend) > 0 {
			fmt.Printf("\n%s\n", bold("Sentiment Trend:"))
			for _, s := range resp.SentimentTrend {
				fmt.Printf("  %s %s %s\n", s.Date, sentimentIcon(s.Sentiment), s.Sentiment)
			}
		}

		if len(resp.RecentReports) > 0 {
			fmt.Printf("\n%s\n", bold("Recent Reports:"))
			for _, r := range resp.RecentReports {
				fmt.Printf("  %s  %s\n", dim(r.Date), r.Summary)
			}
		}
	},
}
