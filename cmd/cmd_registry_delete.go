package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
)

type registryTableModel struct {
	registries []registryInfo
	cursor     int
	selected   bool
	cancelled  bool
}

type registryInfo struct {
	location   string
	status     string
	platforms  int
	components int
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
	// Base style for the container
	baseStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#7a458f")).
		Padding(0, 1)

	// Header style
	headerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("#7a458f")).
		BorderBottom(true).
		Bold(false)

	// Selected row style (white text on purple background)
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("#7a458f")).
		Bold(false)

	// Normal row style
	normalStyle := lipgloss.NewStyle()

	var s strings.Builder

	// Title
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Select a registry to remove:"))
	s.WriteString("\n\n")

	// Calculate dynamic registry column width
	maxRegistryWidth := len("Registry")
	for _, reg := range m.registries {
		if len(reg.location) > maxRegistryWidth {
			maxRegistryWidth = len(reg.location)
		}
	}
	// Cap the registry column width at reasonable limit
	if maxRegistryWidth > 50 {
		maxRegistryWidth = 50
	}

	// Build table content
	var tableContent strings.Builder

	// Header row
	headerRow := fmt.Sprintf("%-*s │ %-15s │ %-10s │ %-12s",
		maxRegistryWidth, "Registry", "Status", "Platforms", "Components")
	tableContent.WriteString(headerStyle.Render(headerRow))
	tableContent.WriteString("\n")

	// Data rows
	for i, reg := range m.registries {
		rowStyle := normalStyle
		if m.cursor == i {
			rowStyle = selectedStyle
		}

		// Truncate registry location if too long
		registryDisplay := reg.location
		if len(registryDisplay) > maxRegistryWidth {
			registryDisplay = "..." + registryDisplay[len(registryDisplay)-(maxRegistryWidth-3):]
		}

		row := fmt.Sprintf("%-*s │ %-15s │ %-10d │ %-12d",
			maxRegistryWidth, registryDisplay, reg.status, reg.platforms, reg.components)
		tableContent.WriteString(rowStyle.Render(row))
		tableContent.WriteString("\n")
	}

	// Apply base style to entire table
	s.WriteString(baseStyle.Render(tableContent.String()))
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("↑/↓: navigate • enter/space: select • q/esc: cancel"))

	return s.String()
}

func selectRegistryInteractively(c *cli.Context, globalConfig *config.GlobalConfig) (string, error) {
	// Create cache to get registry information
	cache, err := registry.NewCache(c.String("cache-dir"), globalConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create cache: %w", err)
	}
	cache.Verbose = c.Bool("verbose")

	// Get current directory for resolving relative paths
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting current directory: %w", err)
	}

	// Build registry info
	var registries []registryInfo
	for location := range globalConfig.Registries {
		info := registryInfo{
			location:   location,
			status:     "Unknown",
			platforms:  0,
			components: 0,
		}

		// Try to get registry info
		reg, err := cache.GetRegistry(location, "", cwd)
		if err != nil {
			info.status = "Error"
		} else {
			info.status = "OK"
			components, err := reg.ScanComponents()
			if err == nil {
				info.platforms = len(components)
				for _, platformComponents := range components {
					info.components += len(platformComponents)
				}
			}
		}

		registries = append(registries, info)
	}

	// Sort registries by location for consistent ordering
	sort.Slice(registries, func(i, j int) bool {
		return registries[i].location < registries[j].location
	})

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

	return result.registries[result.cursor].location, nil
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
