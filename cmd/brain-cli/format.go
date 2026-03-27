package main

import (
	"fmt"
	"strings"
)

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
)

func bold(s string) string   { return ansiBold + s + ansiReset }
func dim(s string) string    { return ansiDim + s + ansiReset }
func red(s string) string    { return ansiRed + s + ansiReset }
func green(s string) string  { return ansiGreen + s + ansiReset }
func yellow(s string) string { return ansiYellow + s + ansiReset }
func cyan(s string) string   { return ansiCyan + s + ansiReset }

func statusIcon(status string) string {
	switch strings.ToLower(status) {
	case "critical", "high", "overdue", "danger":
		return red("✗")
	case "warning", "medium", "behind":
		return yellow("⚠")
	case "ok", "good", "on_track", "completed":
		return green("✓")
	default:
		return "·"
	}
}

func sentimentIcon(sentiment string) string {
	switch strings.ToLower(sentiment) {
	case "positive", "happy", "excited":
		return green("●")
	case "negative", "frustrated", "anxious", "stressed":
		return red("●")
	default:
		return yellow("●")
	}
}

// table prints rows with aligned columns. Each row is a slice of strings.
func table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], bold(h))
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		fmt.Printf("%s  ", dim(strings.Repeat("─", widths[i])))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
}
