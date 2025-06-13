package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
	"github.com/urfave/cli/v2"
)

func main() {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	app := cli.NewApp()
	app.Name = "shry"
	app.Usage = "A command line tool to add and share components for generic projects and platforms"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "cache-dir",
			Usage:   "Directory to cache component registries",
			Value:   filepath.Join(home, ".cache", "shry"),
			EnvVars: []string{"SHRY_CACHE_DIR"},
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:  "init",
			Usage: "Initialize a new project",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "registry",
					Usage:   "Git URL of the component registry (e.g. github.com/networkteam/neos-components[@ref])",
					Aliases: []string{"r"},
				},
			},
			Action: func(c *cli.Context) error {
				registryURL := c.String("registry")
				if registryURL == "" {
					fmt.Println("No registry specified")
					// TODO Show selector for available registries
					return nil
				}

				// Split registry URL and reference if present
				parts := strings.Split(registryURL, "@")
				url := parts[0]
				ref := ""
				if len(parts) > 1 {
					ref = parts[1]
				}

				// Create cache
				cache, err := registry.NewCache(c.String("cache-dir"))
				if err != nil {
					return fmt.Errorf("failed to create cache: %w", err)
				}

				// Get current directory as project root
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("getting current directory: %w", err)
				}

				// Get registry
				reg, err := cache.GetRegistry(url, ref, cwd)
				if err != nil {
					return fmt.Errorf("failed to get registry: %w", err)
				}

				fmt.Printf("Using registry: %s", url)
				if ref != "" {
					fmt.Printf(" (ref: %s)", ref)
				}

				_ = reg

				return nil
			},
		},
		{
			Name:  "ls",
			Usage: "List available components from the registry",
			Action: func(c *cli.Context) error {
				// Find and load the nearest project config
				projectConfig, err := config.FindNearestProjectConfig()
				if err != nil {
					return err
				}

				// Create cache
				cache, err := registry.NewCache(c.String("cache-dir"))
				if err != nil {
					return fmt.Errorf("creating cache: %w", err)
				}

				// Get registry
				reg, err := cache.GetRegistry(projectConfig.Registry, "", projectConfig.ProjectDir)
				if err != nil {
					return fmt.Errorf("getting registry: %w", err)
				}

				// Scan components
				components, err := reg.ScanComponents()
				if err != nil {
					return fmt.Errorf("scanning components: %w", err)
				}

				// Print components
				fmt.Printf("Available components for platform %s:\n\n", projectConfig.Platform)
				for _, component := range components {
					if component.Platform == projectConfig.Platform {
						fmt.Printf("%s\n", component.Name)
						if component.Description != "" {
							fmt.Printf("  %s\n", component.Description)
						}
						fmt.Println()
					}
				}

				return nil
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
