package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var kpisCmd = &cobra.Command{
	Use:   "kpis",
	Short: "Show KPI metrics vs targets",
	Run: func(cmd *cobra.Command, args []string) {
		_, client := mustLoadConfig()

		data, err := client.Get("/openclaw/kpis")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var metrics []struct {
			Name       string  `json:"name"`
			Value      float64 `json:"value"`
			Target     float64 `json:"target"`
			Unit       string  `json:"unit"`
			Direction  string  `json:"direction"`
		}

		if err := json.Unmarshal(data, &metrics); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n\n", bold("KPI Dashboard"))

		if len(metrics) == 0 {
			fmt.Println(dim("No KPIs configured."))
			return
		}

		rows := make([][]string, len(metrics))
		for i, m := range metrics {
			pct := 0.0
			if m.Target != 0 {
				pct = m.Value / m.Target * 100
			}

			status := "ok"
			if m.Direction == "higher_is_better" {
				if pct < 80 {
					status = "critical"
				} else if pct < 100 {
					status = "warning"
				}
			} else {
				if pct > 120 {
					status = "critical"
				} else if pct > 100 {
					status = "warning"
				}
			}

			rows[i] = []string{
				statusIcon(status),
				m.Name,
				formatValue(m.Value, m.Unit),
				formatValue(m.Target, m.Unit),
				fmt.Sprintf("%.0f%%", pct),
			}
		}
		table([]string{"", "Name", "Value", "Target", "Status"}, rows)
	},
}

func formatValue(v float64, unit string) string {
	if unit == "$" {
		if v >= 1000 {
			return fmt.Sprintf("$%.0fK", v/1000)
		}
		return fmt.Sprintf("$%.0f", v)
	}
	if v == float64(int(v)) {
		return fmt.Sprintf("%.0f%s", v, unit)
	}
	return fmt.Sprintf("%.1f%s", v, unit)
}
