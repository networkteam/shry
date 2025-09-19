package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize a new project",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "registry",
				Usage:   "Git URL of the component registry (e.g. github.com/networkteam/neos-components[@ref])",
				Aliases: []string{"r"},
			},
			&cli.StringFlag{
				Name:    "platform",
				Usage:   "Platform to use for the project",
				Aliases: []string{"p"},
			},
		},
		Action: func(c *cli.Context) error {
			registryURL := c.String("registry")
			if registryURL == "" {
				fmt.Println("No registry specified")
				// TODO Show selector for available registries
				return nil
			}

			// Load global configuration
			globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
			if err != nil {
				return fmt.Errorf("loading global config: %w", err)
			}

			// Create cache
			cache, err := registry.NewCache(c.String("cache-dir"), globalConfig)
			if err != nil {
				return fmt.Errorf("failed to create cache: %w", err)
			}
			cache.Verbose = c.Bool("verbose")

			var registryLocation string
			var ref string

			registrySpec := c.String("registry")
			if registrySpec != "" {
				// Split registry URL and reference if present
				parts := strings.Split(registrySpec, "@")
				registryLocation = parts[0]
				ref = ""
				if len(parts) > 1 {
					ref = parts[1]
				}
			} else {
				registryLocations, err := globalConfig.RegistryLocations()
				if err != nil {
					return fmt.Errorf("listing registries: %w", err)
				}

				if len(registryLocations) == 0 {
					return fmt.Errorf("no registries configured, you can add new registries with `shry registry add <registry-location>`")
				}

				registryOpts := make([]huh.Option[string], len(registryLocations))
				for i, location := range registryLocations {
					registryOpts[i] = huh.NewOption(location, location)
				}

				err = huh.NewForm(huh.NewGroup(
					huh.NewSelect[string]().
						Title("Select a registry").
						Options(registryOpts...).
						Value(&registryLocation),
					huh.NewNote().
						Description("You can add new registries with \"shry registry add\"."),
				)).Run()

				if err != nil {
					return err
				}
			}

			// Get current directory as project root
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			// Get registry
			reg, err := cache.GetRegistry(registryLocation, ref, cwd)
			if err != nil {
				return fmt.Errorf("failed to get registry: %w", err)
			}

			components, err := reg.ScanComponents()
			if err != nil {
				return fmt.Errorf("scanning components: %w", err)
			}

			platform := c.String("platform")
			if platform != "" {
				if _, exists := components[platform]; !exists {
					return fmt.Errorf("platform %s not found in registry", platform)
				}
			} else {
				// Prompt for platform
				var platforms []string
				for platform := range components {
					platforms = append(platforms, platform)
				}

				err = huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select a platform").
							Options(huh.NewOptions(platforms...)...).
							Value(&platform),
					),
				).Run()
				if err != nil {
					return err
				}
			}

			// Load project config
			projectConfig, err := config.LoadProjectConfig(cwd)
			if errors.Is(err, os.ErrNotExist) {
				projectConfig = &config.ProjectConfig{}
			} else if err != nil {
				return fmt.Errorf("loading project config: %w", err)
			}

			// Update and save project config
			projectConfig.Registry = registryLocation
			projectConfig.Platform = platform

			err = projectConfig.Save()
			if err != nil {
				return fmt.Errorf("saving project config: %w", err)
			}

			return nil
		},
	}
}
