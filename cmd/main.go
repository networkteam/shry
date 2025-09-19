package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/config"
	"github.com/networkteam/shry/registry"
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
		initCommand(),
		componentAddCommand(),
		componentLsCommand(),
		configCommand(),
		registryCommand(),
	}

	if err := app.Run(os.Args); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			os.Exit(0)
		}
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
