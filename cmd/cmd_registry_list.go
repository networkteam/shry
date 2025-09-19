package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
)

func registryListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List configured registries",
		Action: func(c *cli.Context) error {
			// Load global configuration
			globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
			if err != nil {
				return err
			}

			// Create cache
			cache, err := registry.NewCache(c.String("cache-dir"), globalConfig)
			if err != nil {
				return fmt.Errorf("failed to create cache: %w", err)
			}
			cache.Verbose = c.Bool("verbose")

			// Get current directory for resolving relative paths
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			if len(globalConfig.Registries) == 0 {
				return fmt.Errorf("no registries configured, you can add new registries with `shry registry add <registry-name>`")
			}

			// Build registry info
			type registryInfo struct {
				location   string
				status     string
				platforms  int
				components int
			}

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
					if err != nil {
						info.status = "Error"
					} else {
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

			// Calculate dynamic registry column width
			maxRegistryWidth := len("Registry")
			for _, reg := range registries {
				if len(reg.location) > maxRegistryWidth {
					maxRegistryWidth = len(reg.location)
				}
			}
			// Cap the registry column width at reasonable limit
			if maxRegistryWidth > 50 {
				maxRegistryWidth = 50
			}

			// Display table
			var s strings.Builder

			// Title
			s.WriteString(config.TitleStyle.Render("Configured registries:"))
			s.WriteString("\n")

			// Build table content
			var tableContent strings.Builder

			// Header row
			headerRow := fmt.Sprintf("%-*s │ %-10s │ %-10s │ %-12s",
				maxRegistryWidth, "Registry", "Status", "Platforms", "Components")
			tableContent.WriteString(config.HeaderStyle.Render(headerRow))
			tableContent.WriteString("\n")

			// Data rows
			for _, reg := range registries {
				// Truncate registry location if too long
				registryDisplay := reg.location
				if len(registryDisplay) > maxRegistryWidth {
					registryDisplay = "..." + registryDisplay[len(registryDisplay)-(maxRegistryWidth-3):]
				}

				row := fmt.Sprintf("%-*s │ %-10s │ %-10d │ %-12d",
					maxRegistryWidth, registryDisplay, reg.status, reg.platforms, reg.components)
				tableContent.WriteString(config.NormalStyle.Render(row))
				tableContent.WriteString("\n")
			}

			// Apply base style to entire table
			s.WriteString(config.BaseStyle.Render(tableContent.String()))

			fmt.Print(s.String())
			return nil
		},
	}
}
