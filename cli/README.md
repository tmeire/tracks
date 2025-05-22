# Tracks CLI Module

This module provides a command-line interface for the Tracks application using [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper).

## Overview

The CLI module is designed to provide a flexible and extensible command-line interface for the Tracks application. It uses Cobra for command structure and Viper for configuration management.

## Usage

To use the CLI, build the application and run it with the desired command:

```bash
# Build the CLI application
go build -o floral ./cmd/floral

# Run the CLI application
./floral [command]
```

## Available Commands

### version

Displays the current version of the Tracks application.

```bash
./floral version
```

### tenant

Manages tenants in the multitenancy system.

```bash
./tracks tenant [command]
```

#### Subcommands

##### create

Creates a new tenant with the specified name and subdomain.

```bash
./tracks tenant create [name] [subdomain]
```

### server

Starts the Tracks server with the specified configuration.

```bash
./floral server [flags]
```

#### Flags

- `--port, -p`: Port to run the server on (default: 8080)
- `--host, -H`: Host to bind the server to (default: localhost)
- `--debug, -d`: Enable debug mode (default: false)
- `--config`: Config file (default: $HOME/.floral.yaml)

## Configuration

The CLI uses Viper for configuration management, which supports:

1. Command-line flags
2. Environment variables
3. Configuration files

### Configuration File

By default, the CLI looks for a configuration file named `.floral.yaml` in the user's home directory. You can specify a different configuration file using the `--config` flag.

### Environment Variables

The CLI also reads configuration from environment variables. For example:

- `PORT`: Port to run the server on
- `HOST`: Host to bind the server to
- `DEBUG`: Enable debug mode

## Extending the CLI

To add a new command to the CLI, create a new file in the `internal/tracks/cli` directory and define your command using Cobra. Then, add your command to the root command in the `init()` function.

Example:

```go
package cli

import (
    "github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
    Use:   "new",
    Short: "A brief description of your command",
    Run: func(cmd *cobra.Command, args []string) {
        // Your command logic here
    },
}

func init() {
    rootCmd.AddCommand(newCmd)
}
```
