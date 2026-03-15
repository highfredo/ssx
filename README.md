# ssx — SSH Manager TUI

A terminal user interface for managing SSH connections and tunnels

---

## Features

- 📋 **Host list** — browse all hosts from `~/.ssh/config` with fuzzy search
- 🔌 **Connect** — open an interactive SSH shell directly from the TUI
- 🚇 **Tunnel management** — open/close individual `Forwards` tunnels per host
- 🏷️ **Tags** — annotate hosts with custom colored tags via `#>tags:` comments
- 🔐 **Password support** — inline passwords or external commands via `#>password:` / `#>passwordCommand:`
- 🖥️ **SFTP client launcher** — open WinSCP or FileZilla for the selected host
- 🔑 **SSH copy-id** — copy your public key to a remote host with confirmation prompt
- 🎨 **Themeable** — fully customizable color palette in `~/.config/ssx/config.yaml`

---

## Install

### Linux / macOS

```sh
curl -fsSL https://raw.githubusercontent.com/highfredo/ssx/main/scripts/install.sh | bash
```

Installs to `~/.local/bin` by default. Override with `INSTALL_DIR`:

```sh
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/highfredo/ssx/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/highfredo/ssx/main/scripts/install.ps1 | iex
```

Installs to `%USERPROFILE%\.local\bin` by default. Override with `$env:INSTALL_DIR`:

```powershell
$env:INSTALL_DIR = "C:\Tools"
irm https://raw.githubusercontent.com/highfredo/ssx/main/scripts/install.ps1 | iex
```

### Build from source

```sh
git clone https://github.com/highfredo/ssx
cd ssx
make build        # output: bin/ssx
```

---

## Configuration

Edit `~/.config/ssx/config.yaml` to configure theme and other options.

---

## Key Bindings

### Host List (main screen)

| Key | Action |
|-----|--------|
| `↑ / ↓` | Navigate hosts |
| `enter` | Connect (open SSH shell) |
| `ctrl+a` | Copy SSH public key to host (`ssh-copy-id`) |
| `ctrl+t` / `tab` | Open tunnel view for selected host |
| `ctrl+o` | Open view with all active tunnels |
| `ctrl+x` | Launch SFTP client (WinSCP / FileZilla) |
| `ctrl+g` | Show host info |
| `?` | Toggle full help |
| `ctrl+c` | Quit |

### Tunnel View

| Key | Action |
|-----|--------|
| `↑ / ↓` | Navigate tunnels |
| `enter` | Toggle tunnel open/close |
| `ctrl+x` | Open tunnel URL in browser |
| `esc` / `tab` | Back to host list |
| `ctrl+c` | Quit |

---

## SSH Config Example

```sshconfig
Host example.com
    #>tags: example, myserver
    #>password: mysecretpassword
    HostName 10.0.0.5
    User ubuntu
    IdentityFile ~/.ssh/id_dev
    LocalForward 8080 localhost:80 # app web
    LocalForward 5432 db.internal:5432

Host other
    #>tags: work, critical
    #>passwordCommand: bw get password example.com
    HostName prod.example.com
    User deploy
    Port 2222
```

### Special comments

| Comment | Description |
|---------|-------------|
| `#>tags: tag1, tag2` | Assign colored tags to the host |
| `#>password: secret` | Inline password (used for SSH and ssh-copy-id) |
| `#>passwordCommand: cmd` | Shell command whose stdout is used as password |

