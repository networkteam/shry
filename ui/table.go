package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TableOptions configures how a table is formatted
type TableOptions struct {
	Title        string
	IncludeTitle bool
	RowStyleFunc func(rowIndex int, isSelected bool) lipgloss.Style
}

// FormatTable generates a formatted table string with dynamic column widths
func FormatTable(headers []string, rows [][]string, columnWidths []int, options TableOptions) string {
	if len(rows) == 0 {
		return ""
	}

	var output strings.Builder

	// Add title if requested
	if options.IncludeTitle && options.Title != "" {
		output.WriteString(TitleStyle.Render(options.Title))
		output.WriteString("\n")
	}

	// Build table content
	var tableContent strings.Builder

	// Header row
	var headerParts []string
	for i, header := range headers {
		width := columnWidths[i]
		// Pad all columns for consistent formatting
		headerParts = append(headerParts, fmt.Sprintf("%-*s", width, header))
	}
	headerRow := strings.Join(headerParts, " │ ")
	tableContent.WriteString(HeaderStyle.Render(headerRow))
	tableContent.WriteString("\n")

	// Data rows
	for i, row := range rows {
		// Determine row style
		rowStyle := NormalStyle
		if options.RowStyleFunc != nil {
			rowStyle = options.RowStyleFunc(i, false) // isSelected determined by the function
		}

		var rowParts []string
		for j, cell := range row {
			width := columnWidths[j]
			// Pad all columns to ensure consistent background highlighting
			rowParts = append(rowParts, fmt.Sprintf("%-*s", width, cell))
		}
		rowText := strings.Join(rowParts, " │ ")
		tableContent.WriteString(rowStyle.Render(rowText))
		tableContent.WriteString("\n")
	}

	// Apply base style to entire table
	output.WriteString(BaseStyle.Render(tableContent.String()))

	return output.String()
}

// DefaultRowStyleFunc returns the default row styling function
func DefaultRowStyleFunc(rowIndex int, isSelected bool) lipgloss.Style {
	return NormalStyle
}

// InteractiveRowStyleFunc returns a row styling function for interactive tables
func InteractiveRowStyleFunc(selectedIndex int) func(int, bool) lipgloss.Style {
	return func(rowIndex int, isSelected bool) lipgloss.Style {
		if rowIndex == selectedIndex {
			return SelectedStyle
		}
		return NormalStyle
	}
}

// TruncateText truncates text to maxWidth, adding "..." at the beginning if too long
func TruncateText(text string, maxWidth int) string {
	if len(text) <= maxWidth {
		return text
	}
	if maxWidth <= 3 {
		return "..."
	}
	return "..." + text[len(text)-(maxWidth-3):]
}
