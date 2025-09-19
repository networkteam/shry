package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func componentLsCommand() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "List available components from the registry",
		Action: func(c *cli.Context) error {
			projectConfig, reg, err := loadProjectAndRegistry(c)
			if err != nil {
				return err
			}

			// Scan components
			components, err := reg.ScanComponents()
			if err != nil {
				return fmt.Errorf("scanning components: %w", err)
			}

			// Print components
			fmt.Printf("Available components for platform %s:\n\n", projectConfig.Platform)
			platformComponents, exists := components[projectConfig.Platform]
			if !exists {
				fmt.Printf("No components found for platform %s\n", projectConfig.Platform)
				return nil
			}
			for name, component := range platformComponents {
				fmt.Printf("%s\n", name)
				if component.Description != "" {
					fmt.Printf("  %s\n", component.Description)
				}
				fmt.Println()
			}

			return nil
		},
	}
}
