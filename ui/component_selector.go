package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/networkteam/shry/config"
)

// ComponentItem represents a component in the selection list
type ComponentItem struct {
	name      string
	component *config.Component
}

func (i ComponentItem) FilterValue() string { return i.name }

func (i ComponentItem) Title() string {
	if i.component.Title != "" {
		return i.component.Title
	}
	return i.name
}

func (i ComponentItem) Description() string {
	var parts []string

	// Add description if available
	if i.component.Description != "" {
		parts = append(parts, i.component.Description)
	}

	// Add category if available
	if i.component.Category != "" {
		parts = append(parts, fmt.Sprintf("Category: %s", i.component.Category))
	}

	return strings.Join(parts, " • ")
}

// ComponentSelectorModel handles the component selection state
type ComponentSelectorModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m ComponentSelectorModel) Init() tea.Cmd {
	return nil
}

func (m ComponentSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(ComponentItem)
			if ok {
				m.choice = string(i.name)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ComponentSelectorModel) View() string {
	if m.choice != "" {
		return NormalStyle.Margin(1, 0, 2, 4).Render(fmt.Sprintf("Selected component: %s", m.choice))
	}
	if m.quitting {
		return NormalStyle.Margin(1, 0, 2, 4).Render("No component selected.")
	}
	return "\n" + m.list.View()
}

// ShowComponentSelector displays an interactive component selection list
func ShowComponentSelector(components map[string]map[string]*config.Component, platform string) (string, error) {
	platformComponents, exists := components[platform]
	if !exists {
		return "", fmt.Errorf("no components found for platform %s", platform)
	}

	// Convert components to list items
	items := make([]list.Item, 0, len(platformComponents))
	for name, component := range platformComponents {
		items = append(items, ComponentItem{
			name:      name,
			component: component,
		})
	}

	const defaultWidth = 80
	const listHeight = 20

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = fmt.Sprintf("Select a component for platform: %s", platform)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = TitleStyle
	l.Styles.PaginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	l.Styles.HelpStyle = HelpStyle

	m := ComponentSelectorModel{list: l}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running component selector: %w", err)
	}

	finalModel := result.(ComponentSelectorModel)
	if finalModel.quitting && finalModel.choice == "" {
		return "", nil // User cancelled
	}

	return finalModel.choice, nil
}
