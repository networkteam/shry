package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
)

func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage global configuration",
		Subcommands: []*cli.Command{
			{
				Name:      "set-auth",
				Usage:     "Set authentication for a registry",
				ArgsUsage: "registry-url",
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
					// Get registry URL
					registryURL := c.Args().First()
					if registryURL == "" {
						return fmt.Errorf("registry URL is required")
					}

					// Load global configuration
					globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
					if err != nil {
						return err
					}

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
				Name:      "remove-auth",
				Usage:     "Remove authentication for a registry",
				ArgsUsage: "registry-url",
				Action: func(c *cli.Context) error {
					// Get registry URL
					registryURL := c.Args().First()
					if registryURL == "" {
						return fmt.Errorf("registry URL is required")
					}

					// Load global configuration
					globalConfig, err := config.LoadGlobalConfig(c.String("global-config"))
					if err != nil {
						return err
					}

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
	}
}
