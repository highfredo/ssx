# ssx — SSH Manager TUI

A terminal user interface for managing SSH connections and tunnels, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss).

---

## Features

| Feature | Detail |
|---|---|
| Host list | Reads every non-wildcard `Host` entry from `~/.ssh/config` |
| Tags | Reads host tags from `#>tags:` comments inside `Host` blocks |
| Tunnel descriptions | Reads inline comments on forwarding directives (e.g. `LocalForward ... # web`) |
| Connect | Press **`enter`** (or **`c`**) — opens interactive SSH without auto-applying tunnel forwards |
| Tunnel view | Press **`t`** — shows all `LocalForward`, `RemoteForward`, `DynamicForward` for a host |
| Open tunnel | Press **`o`** — starts `ssh -N` in the background |
| Password auth | Tunnel password is prompted when needed and persisted to `~/.local/state/ssx/passwords.json` |
| Port conflict handling | Shows process holding a local tunnel port and prompts to kill it before opening |
| Close tunnel | Press **`x`** — kills the background process |
| Live state | Each tunnel shows **● OPEN** or **○ CLOSED** in real time |
| Restart awareness | On startup, ssx restores previously app-opened tunnel PIDs and shows live ones as **OPEN** |
| Debug log | Structured logs written to `/tmp/ssx.log` |

---

## Requirements

- Go 1.22 or later
- `ssh` binary on `$PATH`
- A populated `~/.ssh/config`

---

## Build & Run

```bash
# Clone / enter the repo
cd ssx

# Build
make build          # → bin/ssx

# Run
./bin/ssx

# Or build + run in one step
make run
```

---

## Key Bindings

### Host List (main screen)

| Key | Action |
|---|---|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `/` | Filter hosts |
| (filter text) | Matches host alias, hostname, user, and tags |
| `enter` / `c` | Connect to selected host via SSH |
| `t` | Open tunnel management for selected host |
| `q` | Quit |
| `ctrl+c` | Force quit |

### Tunnel View

| Key | Action |
|---|---|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `/` | Edit tunnel filter |
| `x` | Toggle selected tunnel (open/close) |
| `esc` / `q` | Back to host list |

---

## SSH Config Example

```sshconfig
Host dev-box
    #>tags: work, k8s
    HostName 10.0.0.5
    User ubuntu
    IdentityFile ~/.ssh/id_dev
    LocalForward 8080 localhost:80 # app web
    LocalForward 5432 db.internal:5432
    DynamicForward 1080 # socks proxy

Host prod
    #>tags: work, critical
    HostName prod.example.com
    User deploy
    Port 2222
    RemoteForward 9090 localhost:9090
```

`#>tags:` values must be comma-separated and placed under a `Host` block, for example:

```sshconfig
Host mini
    #>tags: minipc, personal
    HostName 192.168.10.2
```

Tunnel descriptions can be placed as inline comments on forwarding lines:

```sshconfig
Host mini
    LocalForward 8080 localhost:8080 # web server
```

---

## Architecture

```
ssx/
├── cmd/ssx/
│   └── main.go              Entry point: parse config → create TUI → graceful shutdown
└── internal/
    ├── credentials/
    │   └── store.go         Password storage (JSON file with 0600 permissions)
    ├── sshconfig/
    │   └── parser.go        Parses ~/.ssh/config → []Host with []Tunnel
    ├── tunnel/
    │   └── manager.go       Manager interface: Open/Close ssh -N processes, track state
    └── ui/
        ├── app.go           Root Bubble Tea model — navigation & message routing
        ├── hostlist.go      Host selection screen (bubbles/list)
        ├── tunnelview.go    Tunnel management screen (custom renderer)
        ├── messages.go      Bubble Tea message types
        └── styles.go        Lipgloss colour palette & style definitions
```

### Key design decisions

- **`tea.ExecProcess`** is used for SSH connections so the TUI yields the terminal cleanly and restores on exit.
- **`ssh -N`** background processes handle tunnels; each has its own `context.CancelFunc` so `Close()` is a clean kill.
- **Dispatch callback** (`manager.SetDispatch`) decouples the tunnel manager from Bubble Tea — the manager sends `any` and main wires it to `p.Send`.
- **Live state** in `TunnelView.View()` is read directly from `manager.State()` on every render frame — no stale copies.

---

## Debugging

```bash
tail -f /tmp/ssx.log
```
