package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Show active alerts for missed check-ins",
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		data, err := client.Get("/openclaw/alerts")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			Alerts []struct {
				EmployeeName string `json:"employee_name"`
				MissedDays   int    `json:"missed_days"`
				Severity     string `json:"severity"`
			} `json:"alerts"`
			Total int `json:"total"`
		}

		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s — %d active\n\n", bold("Alerts"), resp.Total)

		if len(resp.Alerts) == 0 {
			fmt.Println(green("No active alerts. All employees are checking in."))
			return
		}

		rows := make([][]string, len(resp.Alerts))
		for i, a := range resp.Alerts {
			rows[i] = []string{
				statusIcon(a.Severity),
				a.Severity,
				a.EmployeeName,
				fmt.Sprintf("%d days", a.MissedDays),
			}
		}
		table([]string{"", "Severity", "Employee", "Missed"}, rows)
	},
}
