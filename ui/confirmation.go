package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmationOptions configures the confirmation dialog
type ConfirmationOptions struct {
	Title       string
	Description string
	YesText     string
	NoText      string
	DefaultYes  bool
}

// confirmationModel handles the confirmation dialog state
type confirmationModel struct {
	options   ConfirmationOptions
	confirmed bool
	cancelled bool
	cursor    int // 0 = Yes, 1 = No
}

// NewConfirmation creates a new confirmation dialog with default options
func NewConfirmation(title string) ConfirmationOptions {
	return ConfirmationOptions{
		Title:       title,
		Description: "",
		YesText:     "Yes",
		NoText:      "No",
		DefaultYes:  false,
	}
}

// WithDescription adds a description to the confirmation dialog
func (c ConfirmationOptions) WithDescription(desc string) ConfirmationOptions {
	c.Description = desc
	return c
}

// WithYesText customizes the Yes button text
func (c ConfirmationOptions) WithYesText(text string) ConfirmationOptions {
	c.YesText = text
	return c
}

// WithNoText customizes the No button text
func (c ConfirmationOptions) WithNoText(text string) ConfirmationOptions {
	c.NoText = text
	return c
}

// WithDefaultYes sets Yes as the default selection
func (c ConfirmationOptions) WithDefaultYes() ConfirmationOptions {
	c.DefaultYes = true
	return c
}

// ShowConfirmation displays a confirmation dialog and returns the user's choice
func ShowConfirmation(options ConfirmationOptions) (bool, error) {
	cursor := 1 // Default to No
	if options.DefaultYes {
		cursor = 0 // Default to Yes
	}

	model := confirmationModel{
		options:   options,
		confirmed: false,
		cancelled: false,
		cursor:    cursor,
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	result := finalModel.(confirmationModel)
	if result.cancelled {
		return false, nil
	}

	return result.confirmed, nil
}

func (m confirmationModel) Init() tea.Cmd {
	return nil
}

func (m confirmationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			return m, tea.Quit
		case "left", "h":
			m.cursor = 0 // Yes
		case "right", "l":
			m.cursor = 1 // No
		case "tab":
			m.cursor = 1 - m.cursor // Toggle between options
		case "enter", " ":
			m.confirmed = (m.cursor == 0)
			return m, tea.Quit
		case "y", "Y":
			m.confirmed = true
			return m, tea.Quit
		case "n", "N":
			m.confirmed = false
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmationModel) View() string {
	var s strings.Builder

	// Title
	s.WriteString(TitleStyle.Render(m.options.Title))
	s.WriteString("\n")

	// Description (if provided)
	if m.options.Description != "" {
		s.WriteString("\n")
		s.WriteString(HelpStyle.Render(m.options.Description))
		s.WriteString("\n")
	}

	s.WriteString("\n")

	// Buttons horizontally stacked
	var buttons strings.Builder

	// Yes button
	yesStyle := NormalStyle.Padding(0, 2).
		Background(PrimaryDisabledColor).
		Foreground(SecondaryColor)
	if m.cursor == 0 {
		yesStyle = yesStyle.
			Background(PrimaryColor).
			Bold(true)
	}

	buttons.WriteString(yesStyle.Render(m.options.YesText))
	buttons.WriteString("  ")

	// No button
	noStyle := NormalStyle.Padding(0, 2).
		Background(PrimaryDisabledColor).
		Foreground(SecondaryColor)
	if m.cursor == 1 {
		noStyle = noStyle.
			Background(PrimaryColor).
			Bold(true)
	}

	buttons.WriteString(noStyle.Render(m.options.NoText))

	// Center the buttons
	s.WriteString(lipgloss.NewStyle().Width(60).Align(lipgloss.Center).Render(buttons.String()))
	s.WriteString("\n\n")

	// Help text
	s.WriteString(HelpStyle.Render("←/→: navigate • enter/space: select • y/n: quick select • q/esc: cancel"))

	return s.String()
}
