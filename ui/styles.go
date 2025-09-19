package ui

import "github.com/charmbracelet/lipgloss"

// Shared color palette
var (
	PrimaryColor         = lipgloss.Color("#7a458f")
	PrimaryDisabledColor = lipgloss.Color("#41254C")
	SecondaryColor       = lipgloss.Color("#ffffff")
	BorderColor          = lipgloss.Color("#7a458f")
	ErrorColor           = lipgloss.Color("1")
	SuccessColor         = lipgloss.Color("2")
	WarningColor         = lipgloss.Color("3")
	CyanColor            = lipgloss.Color("6")
)

// Shared styles for consistent UI
var (
	// Base container style with border
	BaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	// Header style for tables
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(BorderColor).
			BorderBottom(true).
			Bold(false)

	// Selected row style (white text on purple background)
	SelectedStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Background(PrimaryColor).
			Bold(false)

	// Normal row style
	NormalStyle = lipgloss.NewStyle()

	// Title style
	TitleStyle = lipgloss.NewStyle().Bold(true)

	// Help text style
	HelpStyle = lipgloss.NewStyle().Faint(true)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(ErrorColor).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
)
