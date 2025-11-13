# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GASMS (Grove AppStakes Management System) is a terminal-based TUI application built in Go that helps Pocket Network gateways manage application staking balances. The application displays real-time information about staked applications, their service IDs, and gateway assignments.

**Design Philosophy**: This application is heavily inspired by [k9s](https://github.com/derailed/k9s) - the popular Kubernetes TUI management tool. Our goal is to replicate k9s's intuitive UX patterns, keyboard shortcuts, and visual design for the Pocket Network ecosystem. When developing new features or making UX decisions, k9s should be the primary reference for interaction patterns, navigation flows, and terminal interface design.

## Development Commands

### Building and Running
- `make build` - Build binary for current platform
- `make build-all` - Build binaries for all platforms (Linux, macOS, Windows)
- `make run` - Run the application directly with `go run`
- `make dev` - Run in development mode with live reload (requires `air`)

### Code Quality
- `make fmt` - Format code using `go fmt`
- `make lint` - Lint code (requires `golangci-lint`)
- `make test` - Run tests

### Dependencies
- `make deps` - Install and tidy Go dependencies
- `make clean` - Clean build directory

### Installation
- `make install` - Install binary to `$GOPATH/bin`

## Architecture

### Core Components

1. **main.go** - Primary TUI application using Bubbletea framework
   - Implements state machine with loading, table, command, search, and network selection states
   - Handles all UI rendering and user input
   - Manages application lifecycle and state transitions

2. **config.go** - Configuration management
   - Loads YAML configuration from `config.yaml`
   - Defines `Config` and `Network` structs for multi-network support
   - Networks contain RPC endpoints, gateways, and application lists

3. **pocket.go** - Pocket Network integration
   - `QueryApplications()` function calls `pocketd` CLI to fetch application data
   - Parses JSON responses and filters by gateway addresses
   - Converts stake amounts from raw values to POKT (divides by 1,000,000)

### State Management

The application uses a state machine with these states:
- `stateLoading` - Initial boot with splash screen
- `stateTable` - Main application table view
- `stateCommand` - Command input mode (`:` prefix)
- `stateSearch` - Search mode (`/` prefix) 
- `stateNetworkSelect` - Network selection dialog

### UI Framework

Built with Charm's Bubbletea (TUI framework) and Lipgloss (styling):
- **k9s-inspired keybindings**: Vi-style navigation (`j/k`, `g/G`, `/`, `:`) following k9s conventions
- **k9s-style interface**: Responsive terminal with header bar, status line, and table views
- **k9s color scheme**: Color-coded displays with selection highlighting matching k9s aesthetics
- **k9s navigation patterns**: Command mode, search mode, and context switching similar to k9s

### External Dependencies

- **pocketd** - Pocket Network CLI tool (must be installed separately)
- **air** - Live reload for development (optional)
- **golangci-lint** - Code linting (optional)

## Configuration

The application expects a `config.yaml` file in the working directory with this structure:

```yaml
config:
  # Optional: Configure keyring backend (defaults to pocketd's default)
  keyring-backend: "test"  # Options: os, file, test, kwallet, pass, keychain, memory

  # Optional: Configure pocketd home directory (defaults to $HOME/.pocket)
  pocketd-home: "/custom/path/to/.pocket"

  networks:
    main:
      rpc_endpoint: "http://your-rpc-endpoint"
      bank: "bank_wallet_address_for_fees_and_stake"
      gateways:
        - "gateway_address_1"
      applications:
        - "app1"
    beta:
      rpc_endpoint: "http://beta-rpc-endpoint"
      bank: "bank_wallet_address_for_fees_and_stake"
      gateways:
        - "gateway_address_2"
      applications:
        - "app2"
```

## Key Features

- Multi-network support (switch between main/beta networks)
- Real-time application data fetching via `pocketd` CLI
- Vi-style keyboard navigation
- Search functionality across addresses and service IDs
- Upstake functionality - increase application stakes using hotkey `u` or `:u` command
- Automatic stake amount conversion from raw values to POKT
- Centralized wallet management - all fees and stakes paid from configured bank address
- Responsive terminal interface with proper window sizing

## Development Notes

- The application requires `pocketd` to be installed and accessible in PATH
- Splash screen and logo text are loaded from `art/` directory
- Binary builds are cross-platform (Linux, macOS, Windows)
- Uses Go 1.24+ with modules enabled
- No external runtime dependencies beyond the `pocketd` CLI tool

## UX Reference Guidelines

When implementing new features or improving existing ones, always reference k9s for:
- **Keyboard shortcuts and navigation patterns**
- **Table layouts and data presentation**
- **Color schemes and visual hierarchy**
- **Command/search input handling**
- **Status messages and error display**
- **Loading states and refresh indicators**
- **Help screens and shortcut displays**

The goal is to make GASMS feel familiar to k9s users while adapting the interface for Pocket Network-specific data and workflows.