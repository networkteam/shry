package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
	"github.com/networkteam/shry/template"
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
			Name:      "add",
			Usage:     "Add a component to the project",
			ArgsUsage: "component-name",
			Args:      true,
			Before: func(c *cli.Context) error {
				componentName := c.Args().First()
				if componentName == "" {
					return fmt.Errorf("component name is required")
				}
				return nil
			},
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

				// Lookup the component
				componentName := c.Args().First()
				platformComponents, exists := components[projectConfig.Platform]
				if !exists {
					return fmt.Errorf("no components found for platform %s", projectConfig.Platform)
				}
				component, exists := platformComponents[componentName]
				if !exists {
					return fmt.Errorf("component %s not found for platform %s", componentName, projectConfig.Platform)
				}

				// Verify variables
				resolvedFiles, err := component.ResolveVariables(projectConfig.Variables)
				if err != nil {
					return err
				}

				// Check for existing files
				for _, file := range resolvedFiles {
					dstPath := filepath.Join(projectConfig.ProjectDir, file.Dst)
					if _, err := os.Stat(dstPath); err == nil {
						return fmt.Errorf("destination file already exists: %s", dstPath)
					}
				}

				// Add the component
				fmt.Printf("Adding component %s...\n", componentName)
				for _, file := range resolvedFiles {
					// Read source file
					srcPath := filepath.Join(component.Path, file.Src)
					srcContent, err := reg.ReadFile(srcPath)
					if err != nil {
						return fmt.Errorf("reading source file %s: %w", srcPath, err)
					}

					// Substitute variables in content
					content, err := template.Resolve(string(srcContent), projectConfig.Variables)
					if err != nil {
						return fmt.Errorf("resolving variables in content: %w", err)
					}

					// Create destination directory if needed
					dstPath := filepath.Join(projectConfig.ProjectDir, file.Dst)
					if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
						return fmt.Errorf("creating destination directory: %w", err)
					}

					// Write destination file
					if err := os.WriteFile(dstPath, []byte(content), 0644); err != nil {
						return fmt.Errorf("writing destination file: %w", err)
					}

					fmt.Printf("  Added %s\n", file.Dst)
				}

				return nil
			},
		},
		{
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
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadProjectAndRegistry(c *cli.Context) (*config.ProjectConfig, *registry.Registry, error) {
	// Find and load the nearest project config
	projectConfig, err := config.FindNearestProjectConfig()
	if err != nil {
		return nil, nil, err
	}

	// Build cache
	cache, err := registry.NewCache(c.String("cache-dir"))
	if err != nil {
		return nil, nil, err
	}

	// Get registry for the current project
	reg, err := cache.GetRegistry(projectConfig.Registry, "", projectConfig.ProjectDir)
	if err != nil {
		return nil, nil, err
	}

	return projectConfig, reg, nil
}
