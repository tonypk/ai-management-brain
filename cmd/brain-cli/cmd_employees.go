package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var employeesCmd = &cobra.Command{
	Use:   "employees",
	Short: "List all employees",
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		data, err := client.Get("/openclaw/status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var resp struct {
			TotalEmployees int `json:"total_employees"`
			Submitted      []struct {
				Name string `json:"name"`
			} `json:"submitted"`
			Pending []struct {
				Name string `json:"name"`
			} `json:"pending"`
		}

		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s — %d total\n\n", bold("Employees"), resp.TotalEmployees)

		rows := make([][]string, 0, resp.TotalEmployees)
		for _, e := range resp.Submitted {
			rows = append(rows, []string{e.Name, green("submitted")})
		}
		for _, e := range resp.Pending {
			rows = append(rows, []string{e.Name, yellow("pending")})
		}

		table([]string{"Name", "Status"}, rows)
	},
}
