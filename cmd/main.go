package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mitchellh/go-homedir"
	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
	"github.com/networkteam/shry/template"
	"github.com/sergi/go-diff/diffmatchpatch"
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
		&cli.StringFlag{
			Name:    "global-config",
			Usage:   "Global config path",
			Value:   filepath.Join(home, ".config", "shry", config.GlobalConfigFile),
			EnvVars: []string{"SHRY_GLOBAL_CONFIG"},
		},
		&cli.BoolFlag{
			Name:    "verbose",
			Usage:   "Verbose mode",
			Aliases: []string{"v"},
			EnvVars: []string{"SHRY_VERBOSE"},
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

				_ = reg
				// TODO: Create project config and set registry and platform

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

				componentName := c.Args().First()

				// Resolve component and verify variables
				component, err := reg.ResolveComponent(projectConfig.Platform, componentName)
				if err != nil {
					return err
				}

				// Resolve files and verify variables
				resolvedFiles, err := component.ResolveFiles(projectConfig.Variables)
				if err != nil {
					return err
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
					newContent, err := template.Resolve(string(srcContent), projectConfig.Variables)
					if err != nil {
						return fmt.Errorf("resolving variables in content: %w", err)
					}

					// Check for existing files
					dstPath := filepath.Join(projectConfig.ProjectDir, file.Dst)
					if _, err := os.Stat(dstPath); err == nil {
						existingContent, err := os.ReadFile(dstPath)
						if err != nil {
							return fmt.Errorf("reading existing file: %w", err)
						}

						dmp := diffmatchpatch.New()
						diffs := dmp.DiffMain(string(existingContent), newContent, false)

						// Only show if there are actual changes
						hasChanges := false
						for _, diff := range diffs {
							if diff.Type != diffmatchpatch.DiffEqual {
								hasChanges = true
								break
							}
						}

						if !hasChanges {
							fmt.Printf("  Unchanged %s\n", file.Dst)
							continue
						}

						var choice string
						err = huh.NewForm(
							huh.NewGroup(
								huh.NewSelect[string]().
									Title(fmt.Sprintf("File already exists: %s", file.Dst)).
									Options(
										huh.NewOption("Skip", "skip"),
										huh.NewOption("Overwrite", "overwrite"),
										huh.NewOption("Diff", "diff"),
									).
									Value(&choice),
							),
						).Run()
						if err != nil {
							return err
						}

						if choice == "skip" {
							fmt.Printf("  Skipped %s\n", file.Dst)
							continue
						}
						if choice == "diff" {
							patches := dmp.PatchMake(string(existingContent), diffs)

							fmt.Println("\nDiff:")
							fmt.Println(dmp.PatchToText(patches))

							var choice string
							err = huh.NewForm(
								huh.NewGroup(
									huh.NewSelect[string]().
										Title(fmt.Sprintf("File already exists: %s", file.Dst)).
										Options(
											huh.NewOption("Skip", "skip"),
											huh.NewOption("Overwrite", "overwrite"),
										).
										Value(&choice),
								),
							).Run()
							if err != nil {
								return err
							}

							if choice == "skip" {
								fmt.Printf("  Skipped %s\n", file.Dst)
								continue
							}

							if choice == "overwrite" {
								// Write destination file
								if err := os.WriteFile(dstPath, []byte(newContent), 0644); err != nil {
									return fmt.Errorf("writing destination file: %w", err)
								}

								fmt.Printf("  Overwrite %s\n", file.Dst)

								continue
							}
						}

						// FIXME Check if this is reachable at all
						return fmt.Errorf("destination file already exists: %s", dstPath)
					}

					// Create destination directory if needed
					if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
						return fmt.Errorf("creating destination directory: %w", err)
					}

					// Write destination file
					if err := os.WriteFile(dstPath, []byte(newContent), 0644); err != nil {
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
		{
			Name:  "config",
			Usage: "Manage global configuration",
			Subcommands: []*cli.Command{
				{
					Name:  "set-auth",
					Usage: "Set authentication for a registry",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "registry",
							Usage:    "Registry URL (e.g. github.com/networkteam/neos-components)",
							Required: true,
						},
						&cli.StringFlag{
							Name:  "username",
							Usage: "Username for HTTP authentication",
						},
						&cli.StringFlag{
							Name:  "password",
							Usage: "Password or token for HTTP authentication",
						},
						&cli.StringFlag{
							Name:  "private-key",
							Usage: "Path to private key file for SSH authentication",
						},
						&cli.StringFlag{
							Name:  "key-password",
							Usage: "Password for the private key (if encrypted)",
						},
					},
					Action: func(c *cli.Context) error {
						// Load global configuration
						globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
						if err != nil {
							return err
						}

						// Get registry URL
						registryURL := c.String("registry")

						// Create registry config
						registryConfig := config.RegistryConfig{}

						// Set HTTP authentication if provided
						if username := c.String("username"); username != "" {
							registryConfig.HTTP = &config.HTTPAuth{
								Username: username,
								Password: c.String("password"),
							}
						}

						// Set SSH authentication if provided
						if privateKey := c.String("private-key"); privateKey != "" {
							registryConfig.SSH = &config.SSHAuth{
								PrivateKeyPath: privateKey,
								Password:       c.String("key-password"),
							}
						}

						// Update configuration
						globalConfig.Registries[registryURL] = registryConfig

						// Save configuration
						if err := globalConfig.Save(); err != nil {
							return fmt.Errorf("saving configuration: %w", err)
						}

						fmt.Printf("Authentication configured for registry %s\n", registryURL)
						return nil
					},
				},
				{
					Name:  "remove-auth",
					Usage: "Remove authentication for a registry",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "registry",
							Usage:    "Registry URL",
							Required: true,
						},
					},
					Action: func(c *cli.Context) error {
						// Load global configuration
						globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
						if err != nil {
							return err
						}

						// Get registry URL
						registryURL := c.String("registry")

						// Remove registry configuration
						delete(globalConfig.Registries, registryURL)

						// Save configuration
						if err := globalConfig.Save(); err != nil {
							return fmt.Errorf("saving configuration: %w", err)
						}

						fmt.Printf("Authentication removed for registry %s\n", registryURL)
						return nil
					},
				},
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

	// Load global configuration
	globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
	if err != nil {
		return nil, nil, err
	}

	// Build cache
	cache, err := registry.NewCache(c.String("cache-dir"), globalConfig)
	if err != nil {
		return nil, nil, err
	}
	cache.Verbose = c.Bool("verbose")

	// Get registry for the current project
	reg, err := cache.GetRegistry(projectConfig.Registry, "", projectConfig.ProjectDir)
	if err != nil {
		return nil, nil, err
	}

	return projectConfig, reg, nil
}
