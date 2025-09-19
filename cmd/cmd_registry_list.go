package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/ui"
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

			if len(globalConfig.Registries) == 0 {
				return fmt.Errorf("no registries configured, you can add new registries with `shry registry add <registry-name>`")
			}

			// Collect registry information
			registries, err := ui.CollectRegistryTableInfo(c, globalConfig)
			if err != nil {
				return err
			}

			// Format and display table
			tableOptions := ui.TableOptions{
				Title:        "Configured registries:",
				IncludeTitle: true,
				RowStyleFunc: ui.DefaultRowStyleFunc,
			}

			tableOutput := ui.FormatRegistryTable(registries, tableOptions)
			fmt.Print(tableOutput)
			return nil
		},
	}
}
