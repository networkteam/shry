package main

import "github.com/urfave/cli/v2"

func registryCommand() *cli.Command {
	return &cli.Command{
		Name:  "registry",
		Usage: "Manage component registries",
		Subcommands: []*cli.Command{
			registryAddCommand(),
			registryListCommand(),
			registryDeleteCommand(),
		},
	}
}
