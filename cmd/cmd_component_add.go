package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/urfave/cli/v2"

	"github.com/networkteam/shry/diff"
	"github.com/networkteam/shry/template"
	"github.com/networkteam/shry/ui"
)

func componentAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a component to the project",
		ArgsUsage: "component-name",
		Action: func(c *cli.Context) error {
			projectConfig, reg, err := loadProjectAndRegistry(c)
			if err != nil {
				return err
			}

			componentName := c.Args().First()

			// If no component name provided, show interactive selector
			if componentName == "" {
				// Scan components to show in selector
				components, err := reg.ScanComponents()
				if err != nil {
					return fmt.Errorf("scanning components: %w", err)
				}

				selectedName, err := ui.ShowComponentSelector(components, projectConfig.Platform)
				if err != nil {
					return err
				}

				if selectedName == "" {
					return nil
				}

				componentName = selectedName
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
						lineText1, lineText2, lineArray := dmp.DiffLinesToChars(string(existingContent), newContent)
						diffs := dmp.DiffMain(lineText1, lineText2, false)
						diffs = dmp.DiffCharsToLines(diffs, lineArray)

						diff.PrettyPrint(diffs)

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

						if choice == "overwrite" {
							// Write destination file
							if err := os.WriteFile(dstPath, []byte(newContent), 0644); err != nil {
								return fmt.Errorf("writing destination file: %w", err)
							}

							fmt.Printf("  Overwrite %s\n", file.Dst)

							continue
						}

						if choice == "skip" {
							fmt.Printf("  Skipped %s\n", file.Dst)
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
	}
}
