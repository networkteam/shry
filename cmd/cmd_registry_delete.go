package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/ui"
)

type registryTableModel struct {
	registries []ui.RegistryInfo
	cursor     int
	selected   bool
	cancelled  bool
}

func (m registryTableModel) Init() tea.Cmd {
	return nil
}

func (m registryTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.registries)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m registryTableModel) View() string {
	var s strings.Builder

	// Format table with interactive row styling
	tableOptions := ui.TableOptions{
		Title:        "Select a registry to remove:",
		IncludeTitle: true,
		RowStyleFunc: ui.InteractiveRowStyleFunc(m.cursor),
	}

	// Use shared table formatting but customize for interactive view
	tableOutput := ui.FormatRegistryTable(m.registries, tableOptions)
	s.WriteString(tableOutput)
	s.WriteString("\n")
	s.WriteString(ui.HelpStyle.Render("↑/↓: navigate • enter/space: select • q/esc: cancel"))

	return s.String()
}

func selectRegistryInteractively(c *cli.Context, globalConfig *config.GlobalConfig) (string, error) {
	// Collect registry information using shared function
	registries, err := ui.CollectRegistryTableInfo(c, globalConfig)
	if err != nil {
		return "", err
	}

	// Create and run the table model
	model := registryTableModel{
		registries: registries,
		cursor:     0,
		selected:   false,
		cancelled:  false,
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("running interactive selection: %w", err)
	}

	result := finalModel.(registryTableModel)
	if result.cancelled || !result.selected {
		return "", nil
	}

	return result.registries[result.cursor].Location, nil
}

func registryDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Aliases:   []string{"rm"},
		Usage:     "Remove a registry",
		ArgsUsage: "registry-location",
		Action: func(c *cli.Context) error {
			// Load global configuration
			globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
			if err != nil {
				return err
			}

			if len(globalConfig.Registries) == 0 {
				return fmt.Errorf("no registries configured")
			}

			var registryLocation string

			// Get registry location from args or interactive selection
			if registryLocation = c.Args().First(); registryLocation == "" {
				// Show interactive table for selection
				registryLocation, err = selectRegistryInteractively(c, globalConfig)
				if err != nil {
					return err
				}
				if registryLocation == "" {
					return nil // User cancelled
				}
			}

			// Check if registry exists
			if _, exists := globalConfig.Registries[registryLocation]; !exists {
				return fmt.Errorf("registry %s not found", registryLocation)
			}

			// Confirmation dialog
			var confirm bool
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("Are you sure you want to remove registry '%s'?", registryLocation)).
						Description("This action cannot be undone.").
						Value(&confirm),
				),
			).Run()
			if err != nil {
				return err
			}

			if !confirm {
				fmt.Println("Registry removal cancelled.")
				return nil
			}

			// Remove registry
			delete(globalConfig.Registries, registryLocation)

			// Save configuration
			if err := globalConfig.Save(); err != nil {
				return fmt.Errorf("saving configuration: %w", err)
			}

			fmt.Printf("Removed registry %s\n", registryLocation)
			return nil
		},
	}
}
