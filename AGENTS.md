# ssx Agent Guide

This guide outlines the architecture, patterns, and workflows for working on `ssx`.

## 1. Architecture Overview

`ssx` is a Terminal User Interface (TUI) for managing SSH connections and tunnels.

- **Framework**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (v2).
- **Core Logic**: Wraps the native `ssh` binary for connections and tunnels.
- **Configuration**: Hybird approach using standard `~/.ssh/config` for hosts and `~/.config/ssx/config.yaml` for app settings (themes like `catppuccin`, etc).

### Service Boundaries
- `cmd/ssx`: Entrypoint. Wires up services (`TunnelManager`, `Updater`) and starts the UI.
- `internal/ui`: Contains all Bubble Tea models and view logic.
- `internal/ssh`: Core domain logic. Parses SSH config and manages background `ssh` processes for tunnels.
- `internal/appconfig`: Handles application-level configuration (YAML).

## 2. Key Components & Patterns

### UI Architecture (`internal/ui`)
The UI follows a router-like pattern rooted in `internal/ui/app.go`:
- **Router Model**: `App` struct holds `activePage` (current view) and `rootPage` (host list).
- **Navigation**: Switching views is done by updating `activePage` in `App.Update()`.
- **Modals**: Handled via a separate `modal` field in `App`. Use `base.OpenModalMsg` to trigger them.
- **Components**: Located in subdirectories (`hosts`, `tunnels`, `searchbar`). Each is a self-contained Bubble Tea model.

### SSH & Tunnel Management (`internal/ssh`)
- **Config Parsing**: Uses `github.com/kevinburke/ssh_config` but extends it with custom **Magic Comments** parsing in `config.go`.
  - Example: `#>tags:dev,aws` or `#>password:secret`.
- **Tunnel execution**: Tunnels (`tunnelManager.go`) are spawned as independent `ssh` processes.
  - Flags: `-N` (no remote command), `-o ControlMaster=no` (process isolation).
  - Monitoring: Stderr is piped to detect when forwarding is established.

### Configuration
- **SSH Config**: Primary source of truth for hosts.
- **Styles**: Defined in `internal/styles`. heavily uses `lipgloss`.

## 3. Development Workflow

### Build & Run
- **Run locally**: `make run` (builds and executes `bin/ssx`).
- **Build**: `make build`.
- **Lint/Format**: `make vet`, `make fmt`.
- **Deps**: `make tidy`.

### Testing
- Currently, the project relies on manual testing via the TUI.
- Focus on `internal/ssh` for unit tests if adding logic there.

## 4. Common Tasks

### Adding a new UI View
1. Create a new package under `internal/ui/<view>`.
2. Implement `tea.Model` (Init, Update, View).
3. Define a `Open<View>Msg` in `internal/ui/base/messages.go`.
4. Handle the message in `internal/ui/app.go` to switch `activePage`.

### Modifying SSH Handling
- **Parsing**: Edit `internal/ssh/config.go` to support new directives or magic comments.
- **Execution**: Edit `internal/ssh/tunnelManager.go` or `utils.go` to change how the `ssh` binary is invoked.

## 5. Specific Conventions
- **Magic Comments**: We use `#>` prefix in `~/.ssh/config` to store metadata that `ssh_config` doesn't natively support.
- **State Updates**: Use `tea.Cmd` for side effects (I/O) and `tea.Msg` for state changes. Avoid shared mutable state where possible, except for `TunnelManager` which is passed by reference.

## 6. Language
- Write in English for code comments, commit messages, and documentation to maintain consistency and accessibility for all contributors even if they speak to you in another language.

## 7. OS Support
- Support Linux, macOS, Windows and WSL. Use Go's cross-compilation features and test on all platforms.