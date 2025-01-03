package cmd

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Styles for sections and headers
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7287fd")).
		MarginBottom(1)

	// Table styles
	tableStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#babbf1")).
		MarginBottom(1)

	// Helper functions
	NewTable = func(columns []table.Column, rows []table.Row) table.Model {
		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithHeight(len(rows)),
			table.WithFocused(false),
		)
		return t
	}
)
