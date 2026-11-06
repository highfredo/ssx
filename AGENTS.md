# ssx Agent Guide

This guide outlines the architecture, patterns, and workflows for working on `ssx`.

## 1. Architecture Overview

`ssx` is a Terminal User Interface (TUI) for managing SSH connections and tunnels.

- **Framework**: Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (v2).
- **Core Logic**: Wraps the native `ssh` binary for connections and tunnels.
- **Configuration**: Hybird approach using standard `~/.ssh/config` for hosts and `~/.config/ssx/config.yaml` for app settings (themes like `catppuccin`, etc).

### Service Boundaries
- `cmd/ssx`: Entrypoint. Loads the host list (`ssh.LoadConfig`), handles CLI subcommands (`internal/cli`) and starts the UI with `ui.NewApp(hosts)`.
- `internal/ui`: Contains all Bubble Tea models and view logic.
- `internal/ssh`: Core domain logic. Parses SSH config and manages background `ssh` processes for tunnels. Split by responsibility:
  - `host.go` (`HostConfig`, `Hosts`), `addr.go` (`ParseConnection`), `config.go` + `include.go` + `magic.go` + `forward.go` (config parsing).
  - `tunnel.go` (`Tunnel`, `TunnelStatus`, `ParseTunnel`), `ports.go` (listening-port snapshot via gopsutil), `status.go` (`GetTunnelStatus`, `HostTunnels`, `OpenTunnels`), `lifecycle.go` (`OpenTunnel`, `CloseTunnel`, `KillPortOwner`).
  - `cmd.go` (building `ssh` commands), `keys.go` (public keys, `CopyID`, PuTTY conversion).
  - There is **no global state**: the `ssh.Hosts` slice is loaded once in `main` and passed explicitly to the UI/CLI.
- `internal/appconfig`: Handles application-level configuration (YAML). `appconfig.Load()` is called once in `main`; the result is injected explicitly: `styles.Apply(cfg.Theme)`, `keys := base.NewKeybindings(cfg.Keybindings)`, `ui.NewApp(hosts, cfg, keys)` (which passes `cfg.Tags` and `keys` down to page constructors; pages never read global key state), `ftp.Launch(h, cfg.FTP)`. There is no `appconfig.Get()` singleton.

## 2. Key Components & Patterns

### UI Architecture (`internal/ui`)
The UI follows a router-like pattern rooted in `internal/ui/app.go`:
- **Router Model**: `App` struct holds `activePage` (current view) and `rootPage` (host list).
- **Navigation**: Switching views is done by updating `activePage` in `App.Update()`.
- **Modals**: Handled via a separate `modal` field in `App`. Use `modal.OpenAlert` / `modal.OpenConfirm` / `modal.OpenDangerConfirm` (destructive actions, defaults to "No") / `modal.OpenPassword` (masked input; used to retry after `ssh.ErrAuthFailed`) / `modal.OpenSpinner` (which emit `base.OpenModalMsg`); shared rendering lives in `modal/dialog.go`.
- **Host actions** shared by several pages (copy key, SFTP, edit config) live in `internal/ui/actions.go` and are triggered with the `base.*Msg` messages.
- **Components**: Located in subdirectories (`hosts`, `tunnels`, `searchbar`). Each is a self-contained Bubble Tea model.
- **Lists**: `internal/ui/list` wraps `bubbles/list` with a search bar, an optional virtual item built from the query, a `StatusFunc` counter, an empty-state hint and a compact mode below `list.NarrowWidth` (delegates implementing `Compactable` drop descriptions). Custom delegates decorate the default render and must pass each line through `list.TruncateLine`. Use `list.Filter` (byte→rune match indexes) whenever highlighting matches with `lipgloss.StyleRunes`.

### SSH & Tunnel Management (`internal/ssh`)
- **Config Parsing**: Uses `github.com/kevinburke/ssh_config` but extends it with custom **Magic Comments** parsing in `config.go`.
  - Example: `#>tags:dev,aws` or `#>password:secret`.
- **Tunnel execution**: Tunnels (`lifecycle.go`) are spawned as independent `ssh` processes.
  - Flags: `-N` (no remote command), `-o ControlMaster=no` (process isolation).
  - Monitoring: Stderr is piped to detect when forwarding is established.

### Configuration
- **SSH Config**: Primary source of truth for hosts.
- **Styles**: Defined in `internal/styles`; the palette (`ColorPrimary`, `ColorText`, `ColorMuted`, `ColorWarning`, …) comes from `appconfig.ThemeConfig` via `styles.Apply`. Always read `styles.*` at render time — never capture them in package-level vars, or user themes are ignored. Tags use `styles.ChipStyle`.

## 3. Development Workflow

### Build & Run
- **Run locally**: `mise run run` (builds and executes `bin/ssx`).
- **Build**: `mise run build`.
- **Lint/Format**: `mise run vet`, `mise run fmt`.
- **Deps**: `mise run tidy`.

### Testing
- Currently, the project relies on manual testing via the TUI.
- `internal/ssh` has unit tests (`go test -race ./...`); add table-driven tests there when touching parsing or status logic.

## 4. Common Tasks

### Adding a new UI View
1. Create a new package under `internal/ui/<view>`.
2. Implement `tea.Model` (Init, Update, View).
3. Define a `Open<View>Msg` in `internal/ui/base/messages.go`.
4. Handle the message in `internal/ui/app.go` to switch `activePage`.

### Modifying SSH Handling
- **Parsing**: Edit `internal/ssh/config.go` to support new directives or magic comments.
- **Execution**: Edit `internal/ssh/lifecycle.go` (tunnels) or `internal/ssh/cmd.go` (shell/commands) to change how the `ssh` binary is invoked.

## 5. Specific Conventions
- **Magic Comments**: We use `#>` prefix in `~/.ssh/config` to store metadata that `ssh_config` doesn't natively support.
- **State Updates**: Use `tea.Cmd` for side effects (I/O) and `tea.Msg` for state changes. Avoid shared mutable state; pass `ssh.Hosts` / `*ssh.HostConfig` explicitly instead of using package-level caches.

## 6. Language
- Write in English for code comments, commit messages, and documentation to maintain consistency and accessibility for all contributors even if they speak to you in another language.

## 7. OS Support
- Support Linux, macOS, Windows and WSL. Use Go's cross-compilation features and test on all platforms.