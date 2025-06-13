package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/go-git/go-git/v5/plumbing/transport"
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

				var registryName string
				var ref string

				registrySpec := c.String("registry")
				if registrySpec != "" {
					// Split registry URL and reference if present
					parts := strings.Split(registrySpec, "@")
					registryName = parts[0]
					ref = ""
					if len(parts) > 1 {
						ref = parts[1]
					}
				} else {
					registryNames, err := cache.ListRegistries()
					if err != nil {
						return fmt.Errorf("listing registries: %w", err)
					}

					registryOpts := make([]huh.Option[string], len(registryNames))
					for i, name := range registryNames {
						registryOpts[i] = huh.NewOption(name, name)
					}

					err = huh.NewForm(huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select a registry").
							Options(registryOpts...).
							Value(&registryName),
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
				reg, err := cache.GetRegistry(registryName, ref, cwd)
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

				err = addComponent(projectConfig, reg, componentName, nil)
				if err != nil {
					return err
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

				// Group components by category
				categories := make(map[string][]*config.Component)
				var uncategorized []*config.Component

				for _, component := range platformComponents {
					if component.Category == "" {
						uncategorized = append(uncategorized, component)
					} else {
						categories[component.Category] = append(categories[component.Category], component)
					}
				}

				// Sort category names
				var categoryNames []string
				for category := range categories {
					categoryNames = append(categoryNames, category)
				}
				slices.Sort(categoryNames)

				// Print uncategorized components first
				if len(uncategorized) > 0 {
					fmt.Println("Uncategorized:")
					slices.SortFunc(uncategorized, func(a, b *config.Component) int {
						return strings.Compare(a.Name, b.Name)
					})
					for _, component := range uncategorized {
						fmt.Printf("  %s\n", component.Name)
						if component.Description != "" {
							fmt.Printf("    %s\n", component.Description)
						}
					}
					fmt.Println()
				}

				// Print categorized components
				for _, category := range categoryNames {
					fmt.Printf("%s:\n", category)
					components := categories[category]
					slices.SortFunc(components, func(a, b *config.Component) int {
						return strings.Compare(a.Name, b.Name)
					})
					for _, component := range components {
						fmt.Printf("  %s\n", component.Name)
						if component.Description != "" {
							fmt.Printf("    %s\n", component.Description)
						}
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
		{
			Name:  "registry",
			Usage: "Manage component registries",
			Subcommands: []*cli.Command{
				{
					Name:      "add",
					Usage:     "Add a new registry",
					ArgsUsage: "registry-name",
					Args:      true,
					Flags: []cli.Flag{
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
						// Get registry name
						registryName := c.Args().First()
						if registryName == "" {
							return fmt.Errorf("registry name is required")
						}

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

						// Try to get the registry to verify it's accessible
						reg, err := cache.GetRegistry(registryName, "", cwd)
						if err != nil {
							// Check if authentication is required
							if errors.Is(err, transport.ErrAuthenticationRequired) {
								var authType string
								err = huh.NewForm(
									huh.NewGroup(
										huh.NewSelect[string]().
											Title("Authentication required. Choose authentication method:").
											Options(
												huh.NewOption("HTTP Basic", "http"),
												huh.NewOption("SSH Key", "ssh"),
											).
											Value(&authType),
									),
								).Run()
								if err != nil {
									return err
								}

								registryConfig := config.RegistryConfig{}

								if authType == "http" {
									// Use provided flags or prompt for HTTP auth
									username := c.String("username")
									password := c.String("password")

									if username == "" {
										err = huh.NewForm(
											huh.NewGroup(
												huh.NewInput().
													Title("Username").
													Value(&username),
											),
										).Run()
										if err != nil {
											return err
										}
									}

									if password == "" {
										err = huh.NewForm(
											huh.NewGroup(
												huh.NewInput().
													Title("Password or token").
													Value(&password).
													EchoMode(huh.EchoModePassword),
											),
										).Run()
										if err != nil {
											return err
										}
									}

									registryConfig.HTTP = &config.HTTPAuth{
										Username: username,
										Password: password,
									}
								} else {
									// Use provided flags or prompt for SSH auth
									privateKey := c.String("private-key")
									keyPassword := c.String("key-password")

									if privateKey == "" {
										err = huh.NewForm(
											huh.NewGroup(
												huh.NewInput().
													Title("Path to private key file").
													Value(&privateKey),
											),
										).Run()
										if err != nil {
											return err
										}
									}

									if keyPassword == "" {
										err = huh.NewForm(
											huh.NewGroup(
												huh.NewInput().
													Title("Password for private key (if encrypted)").
													Value(&keyPassword).
													EchoMode(huh.EchoModePassword),
											),
										).Run()
										if err != nil {
											return err
										}
									}

									registryConfig.SSH = &config.SSHAuth{
										PrivateKeyPath: privateKey,
										Password:       keyPassword,
									}
								}

								// Update global config with auth
								if globalConfig.Registries == nil {
									globalConfig.Registries = make(map[string]config.RegistryConfig)
								}
								globalConfig.Registries[registryName] = registryConfig

								// Try again with authentication
								reg, err = cache.GetRegistry(registryName, "", cwd)
								if err != nil {
									return fmt.Errorf("failed to access registry with authentication: %w", err)
								}
							} else {
								return fmt.Errorf("failed to access registry: %w", err)
							}
						}

						// Add registry to global config if not already added
						if globalConfig.Registries == nil {
							globalConfig.Registries = make(map[string]config.RegistryConfig)
						}
						if _, exists := globalConfig.Registries[registryName]; !exists {
							globalConfig.Registries[registryName] = config.RegistryConfig{}
						}

						// Verify we can scan components
						components, err := reg.ScanComponents()
						if err != nil {
							return fmt.Errorf("failed to scan components: %w", err)
						}

						if err := globalConfig.Save(); err != nil {
							return fmt.Errorf("saving configuration: %w", err)
						}

						// Print summary
						fmt.Printf("Added registry %s\n", registryName)
						fmt.Printf("Found components for platforms:\n")
						for platform := range components {
							fmt.Printf("  - %s (%d components)\n", platform, len(components[platform]))
						}

						return nil
					},
				},
				{
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

						fmt.Println("Configured registries:")
						for name := range globalConfig.Registries {
							// Try to get the registry to verify it's accessible
							reg, err := cache.GetRegistry(name, "", cwd)
							if err != nil {
								fmt.Printf("  %s (error: %v)\n", name, err)
								continue
							}

							// Get component count
							components, err := reg.ScanComponents()
							if err != nil {
								fmt.Printf("  %s (error scanning: %v)\n", name, err)
								continue
							}

							// Count total components
							total := 0
							for _, platformComponents := range components {
								total += len(platformComponents)
							}

							fmt.Printf("  %s (%d components across %d platforms)\n", name, total, len(components))
						}

						return nil
					},
				},
				{
					Name:    "remove",
					Aliases: []string{"rm"},
					Usage:   "Remove a registry",
					Flags: []cli.Flag{
						&cli.StringFlag{
							Name:     "name",
							Usage:    "Registry name",
							Required: true,
						},
					},
					Action: func(c *cli.Context) error {
						// Load global configuration
						globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
						if err != nil {
							return err
						}

						// Get registry name
						registryName := c.String("name")

						// Check if registry exists
						if _, exists := globalConfig.Registries[registryName]; !exists {
							return fmt.Errorf("registry %s not found", registryName)
						}

						// Remove registry
						delete(globalConfig.Registries, registryName)

						// Save configuration
						if err := globalConfig.Save(); err != nil {
							return fmt.Errorf("saving configuration: %w", err)
						}

						fmt.Printf("Removed registry %s\n", registryName)
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

func addComponent(projectConfig *config.ProjectConfig, reg *registry.Registry, componentName string, addedComponents []string) error {
	// Skip if already added
	if slices.Contains(addedComponents, componentName) {
		return nil
	}

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

	componentType := "component"
	if len(addedComponents) > 0 {
		componentType = "dependency"
	}

	// Add the component
	fmt.Printf("Adding %s %s...\n", componentType, componentName)
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

	addedComponents = append(addedComponents, componentName)

	for _, dependency := range component.Dependencies {
		err = addComponent(projectConfig, reg, dependency, addedComponents)
		if err != nil {
			return err
		}
	}

	return nil
}
