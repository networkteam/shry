# shry
Cheri, Darling, Schatzi 🍒😘 manage components with style

## Overview
Shry (pronounced `[ˈtʃeri]`) is a cli tool to manage your own component registry for multiple platforms à la [shadcn/ui](https://github.com/shadcn-ui/ui).

## Installation

### Homebrew (macOS)
```bash
brew tap networkteam/tap
brew install shry
```

### Manual Installation
Download the latest release from the [releases page](https://github.com/networkteam/shry/releases) and install it in your PATH.

## Usage

### Initialize a Project
Initialize a new project with a component registry:
```bash
shry init
```
This will:
- Prompt you to select a registry (or use `--registry` to specify one)
- Prompt you to select a platform
- Create a project configuration file

Options:
- `--registry, -r`: Git URL of the component registry (e.g. github.com/networkteam/neos-components[@ref])
- `--platform, -p`: Platform to use for the project

### List Components
List available components from the registry:
```bash
shry ls
```
Components are grouped by category and sorted alphabetically.

### Add Components
Add a component to your project:
```bash
shry add <component-name>
```
This will:
- Add the component and all its dependencies
- Handle file conflicts with options to skip, overwrite, or show diff
- Resolve variables in component files

### Manage Registries

#### Add a Registry
```bash
shry registry add <registry-location>
```
This will:
- Add a new registry to your configuration
- Prompt for authentication if needed
- Verify the registry is accessible

#### List Registries
```bash
shry registry list
```
Shows all configured registries with their component counts.

#### Remove a Registry
```bash
shry registry remove <registry-location>
```

### Authentication

#### Set Authentication
```bash
shry config set-auth <registry-url>
```
Options:
- `--username`: Username for HTTP authentication
- `--password`: Password or token for HTTP authentication
- `--private-key`: Path to private key file for SSH authentication
- `--key-password`: Password for the private key (if encrypted)

#### Remove Authentication
```bash
shry config remove-auth <registry-url>
```

## Configuration

### Global Configuration
The global configuration is stored in:
- macOS: `~/.config/shry/global.yaml`

### Project Configuration
Each project has its own configuration file that stores:
- Selected registry
- Platform
- Project variables

### Environment Variables
- `SHRY_CACHE_DIR`: Directory to cache component registries (default: `~/.cache/shry`)
- `SHRY_GLOBAL_CONFIG`: Global config path
- `SHRY_VERBOSE`: Enable verbose mode

## Component Registry Structure
A component registry is a Git repository containing components. Each component has:
- A `shry.yaml` configuration file
- Source files to be copied
- Optional dependencies on other components
- Optional category for grouping

Example `shry.yaml`:
```yaml
name: my-component
title: My Component
description: A beautiful component
platform: neos
category: Layout
dependencies:
  - base-component
files:
  - src: src/MyComponent.tsx
    dst: Components/MyComponent.tsx
variables:
  color: primary
```
