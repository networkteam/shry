package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
)

func registryAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a new registry",
		ArgsUsage: "registry-location",
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
			registryLocation := c.Args().First()
			if registryLocation == "" {
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

			registryName := registryLocation

			// Try to get the registry to verify it's accessible
			reg, err := cache.GetRegistry(registryLocation, "", cwd)
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
			} else {
				registryName = reg.Name
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
	}
}