package ui

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
)

// RegistryInfo holds information about a registry for table display
type RegistryInfo struct {
	Location   string
	Status     string
	Platforms  int
	Components int
}

// CollectRegistryTableInfo gathers registry information from the global configuration
func CollectRegistryTableInfo(c *cli.Context, globalConfig *config.GlobalConfig) ([]RegistryInfo, error) {
	// Create cache
	cache, err := registry.NewCache(c.String("cache-dir"), globalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	cache.Verbose = c.Bool("verbose")

	// Get current directory for resolving relative paths
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}

	// Build registry info
	var registries []RegistryInfo
	for location := range globalConfig.Registries {
		info := RegistryInfo{
			Location:   location,
			Status:     "Unknown",
			Platforms:  0,
			Components: 0,
		}

		// Try to get registry info
		reg, err := cache.GetRegistry(location, "", cwd)
		if err != nil {
			info.Status = "Error"
		} else {
			info.Status = "OK"
			components, err := reg.ScanComponents()
			if err != nil {
				info.Status = "Error"
			} else {
				info.Platforms = len(components)
				for _, platformComponents := range components {
					info.Components += len(platformComponents)
				}
			}
		}

		registries = append(registries, info)
	}

	// Sort registries by location for consistent ordering
	sort.Slice(registries, func(i, j int) bool {
		return registries[i].Location < registries[j].Location
	})

	return registries, nil
}

// FormatRegistryTable generates a formatted table string from registry information
func FormatRegistryTable(registries []RegistryInfo, options TableOptions) string {
	if len(registries) == 0 {
		return ""
	}

	// Calculate dynamic registry column width
	maxRegistryWidth := len("Registry")
	for _, reg := range registries {
		if len(reg.Location) > maxRegistryWidth {
			maxRegistryWidth = len(reg.Location)
		}
	}
	// Cap the registry column width at reasonable limit
	if maxRegistryWidth > 50 {
		maxRegistryWidth = 50
	}

	// Define headers and column widths
	headers := []string{"Registry", "Status", "Platforms", "Components"}
	columnWidths := []int{maxRegistryWidth, 10, 10, 12}

	// Convert registry data to table rows
	var rows [][]string
	for _, reg := range registries {
		// Truncate registry location if too long
		registryDisplay := TruncateText(reg.Location, maxRegistryWidth)

		row := []string{
			registryDisplay,
			reg.Status,
			strconv.Itoa(reg.Platforms),
			strconv.Itoa(reg.Components),
		}
		rows = append(rows, row)
	}

	return FormatTable(headers, rows, columnWidths, options)
}
