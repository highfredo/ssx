// Package ui implements the root Bubble Tea application model for ssx.
//
// Architecture overview:
//
//	App  (root model — owns navigation state)
//	 ├── HostList   (bubbles/list — SSH host selection)
//	 └── TunnelView (custom list — per-host tunnel management)
//
// Navigation flow:
//
//	HostList  --[t]--> TunnelView
//	TunnelView --[esc/q]--> HostList
//
// SSH connections are executed with tea.ExecProcess, which suspends the TUI,
// hands the terminal to SSH, then restores the TUI when SSH exits.
//
// Tunnels run as background "ssh -N" processes managed by tunnel.Manager.
// Unexpected exits are reported via ExitedMsg dispatched through the callback
// set on the manager (see main.go).
package ui

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/highfredo/ssx/internal/credentials"
	"github.com/highfredo/ssx/internal/sshconfig"
	"github.com/highfredo/ssx/internal/sshpass"
	"github.com/highfredo/ssx/internal/tunnel"
)

// viewState enumerates the top-level screens of the TUI.
type viewState int

const (
	viewHostList  viewState = iota // Main host-selection screen.
	viewTunnelMgr                  // Tunnel management screen for one host.
)

// App is the root Bubble Tea model. It owns top-level navigation and status
// messaging, and delegates rendering/updates to the active sub-model.
type App struct {
	state       viewState
	hostList    HostList
	tunnelView  TunnelView
	tunnelMgr   tunnel.Manager
	hosts       []*sshconfig.Host
	width       int
	height      int
	statusMsg   string
	statusIsErr bool
	passwords   map[string]string
	passwordDB  *credentials.Store
	prompt      *passwordPrompt
}

type pendingAction int

const (
	actionConnect pendingAction = iota
	actionOpenTunnel
)

type passwordPrompt struct {
	input    textinput.Model
	hostName string
	action   pendingAction
	tunnel   sshconfig.Tunnel
}

// NewApp constructs the App model. Call tea.NewProgram(NewApp(...)) and then
// set mgr.SetDispatch after creating the program (see main.go).
func NewApp(hosts []*sshconfig.Host, mgr tunnel.Manager) *App {
	passwordDB := credentials.NewStore()
	passwords, err := passwordDB.Load()
	if err != nil {
		passwords = make(map[string]string)
	}
	return &App{
		state:      viewHostList,
		hostList:   NewHostList(hosts),
		tunnelMgr:  mgr,
		hosts:      hosts,
		passwords:  passwords,
		passwordDB: passwordDB,
	}
}

// Init implements tea.Model. No startup commands are required.
func (a *App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model and routes every message to the correct handler.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.prompt != nil {
		if _, ok := msg.(tea.KeyMsg); !ok {
			// Allow non-input async messages (e.g. tunnel exit notifications)
			// to continue through the normal update flow.
		} else {
			return a.updatePasswordPrompt(msg)
		}
	}

	switch msg := msg.(type) {

	// ── Terminal resize ───────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.hostList.SetSize(msg.Width, msg.Height)
		if a.state == viewTunnelMgr {
			a.tunnelView.SetSize(msg.Width, msg.Height)
		}
		return a, nil

	// ── Hard quit ─────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case requestConnectMsg:
		if host := msg.host; host != nil {
			return a, connectCmd(host.Name, a.passwords[host.Name])
		}

	case requestOpenTunnelMsg:
		if pw := a.passwords[msg.hostName]; pw != "" {
			return a, openTunnelCmd(a.tunnelMgr, msg.hostName, msg.tunnel, pw)
		}
		a.beginPasswordPrompt(msg.hostName, actionOpenTunnel, msg.tunnel)
		return a, nil

	// ── Navigation: open tunnel view ─────────────────────────────────────────
	case openTunnelViewMsg:
		a.tunnelView = NewTunnelView(msg.host, a.tunnelMgr, a.width, a.height)
		a.state = viewTunnelMgr
		a.clearStatus()
		return a, nil

	// ── Navigation: back to host list ────────────────────────────────────────
	case backMsg:
		a.state = viewHostList
		a.clearStatus()
		return a, nil

	// ── SSH connection exited ────────────────────────────────────────────────
	case SSHExitedMsg:
		if msg.Err != nil {
			a.setError(fmt.Sprintf("SSH exited: %v", msg.Err))
		} else {
			a.setOK("SSH session ended.")
		}
		return a, nil

	// ── Tunnel open result ────────────────────────────────────────────────────
	case openTunnelResultMsg:
		if msg.Err != nil {
			a.setError(fmt.Sprintf("open tunnel: %v", msg.Err))
		} else {
			a.setOK(fmt.Sprintf("Tunnel opened  (%s)", msg.TunnelID))
		}
		return a, a.maybeRefresh()

	// ── Tunnel close result ───────────────────────────────────────────────────
	case closeTunnelResultMsg:
		if msg.Err != nil {
			a.setError(fmt.Sprintf("close tunnel: %v", msg.Err))
		} else {
			a.setOK("Tunnel closed.")
		}
		return a, a.maybeRefresh()

	// ── Unexpected tunnel exit (dispatched by tunnel.manager goroutine) ───────
	case tunnel.ExitedMsg:
		a.setError(fmt.Sprintf("Tunnel exited unexpectedly: %s", msg.TunnelID))
		return a, a.maybeRefresh()

	// ── Refresh re-render request ─────────────────────────────────────────────
	case refreshTunnelViewMsg:
		// Nothing to update in state — View() pulls live data from the manager.
		return a, nil
	}

	// ── Delegate to the active sub-model ─────────────────────────────────────
	switch a.state {
	case viewHostList:
		newHL, cmd := a.hostList.Update(msg)
		a.hostList = newHL
		return a, cmd

	case viewTunnelMgr:
		newTV, cmd := a.tunnelView.Update(msg)
		a.tunnelView = newTV
		return a, cmd
	}

	return a, nil
}

// View implements tea.Model and renders the active screen with an optional
// status bar appended at the bottom.
func (a *App) View() string {
	var body string
	switch a.state {
	case viewHostList:
		body = a.hostList.View()
	case viewTunnelMgr:
		body = a.tunnelView.View()
	}

	if a.prompt != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, a.renderPasswordPrompt())
	}

	if a.statusMsg == "" {
		return body
	}

	var statusBar string
	if a.statusIsErr {
		statusBar = errorStyle.Render("✗ " + a.statusMsg)
	} else {
		statusBar = statusOKStyle.Render("✓ " + a.statusMsg)
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (a *App) setError(msg string) {
	a.statusMsg = msg
	a.statusIsErr = true
}

func (a *App) setOK(msg string) {
	a.statusMsg = msg
	a.statusIsErr = false
}

func (a *App) clearStatus() {
	a.statusMsg = ""
	a.statusIsErr = false
}

func (a *App) beginPasswordPrompt(hostName string, action pendingAction, t sshconfig.Tunnel) {
	in := textinput.New()
	in.Placeholder = "SSH password"
	in.EchoMode = textinput.EchoPassword
	in.EchoCharacter = '•'
	in.Prompt = "Password: "
	in.Focus()
	a.prompt = &passwordPrompt{
		input:    in,
		hostName: hostName,
		action:   action,
		tunnel:   t,
	}
	a.clearStatus()
}

func (a *App) updatePasswordPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return a, tea.Quit
		case "esc":
			a.prompt = nil
			a.setError("Password prompt cancelled.")
			return a, nil
		case "enter":
			pw := a.prompt.input.Value()
			hostName := a.prompt.hostName
			action := a.prompt.action
			t := a.prompt.tunnel
			a.prompt = nil
			if pw == "" {
				a.setError("Password cannot be empty.")
				return a, nil
			}
			a.passwords[hostName] = pw
			if err := a.passwordDB.Save(a.passwords); err != nil {
				a.setError(fmt.Sprintf("save password: %v", err))
				return a, nil
			}
			switch action {
			case actionConnect:
				return a, connectCmd(hostName, pw)
			case actionOpenTunnel:
				return a, openTunnelCmd(a.tunnelMgr, hostName, t, pw)
			}
		}
	}

	var cmd tea.Cmd
	a.prompt.input, cmd = a.prompt.input.Update(msg)
	return a, cmd
}

func (a *App) renderPasswordPrompt() string {
	if a.prompt == nil {
		return ""
	}
	title := fmt.Sprintf(" Password for tunnel on %s ", a.prompt.hostName)
	help := helpStyle.Render("[enter] submit  [esc] cancel")
	content := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(title), a.prompt.input.View(), help)
	return panelStyle.Render(content)
}

// maybeRefresh returns a refreshTunnelViewMsg command when the tunnel screen
// is active, so it re-renders with the latest state from the manager.
func (a *App) maybeRefresh() tea.Cmd {
	if a.state == viewTunnelMgr {
		return func() tea.Msg { return refreshTunnelViewMsg{} }
	}
	return nil
}

// ── Tea commands ──────────────────────────────────────────────────────────────

// connectCmd suspends the TUI and opens an interactive SSH session.
// The TUI is restored automatically when the session ends.
func connectCmd(hostName, password string) tea.Cmd {
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		sshPath = "ssh"
	}
	// ClearAllForwardings prevents Local/Remote/DynamicForward directives
	// from auto-applying during interactive connections.
	cmd := exec.Command(sshPath, "-o", "ClearAllForwardings=yes", hostName) //nolint:gosec
	cleanup, err := sshpass.Configure(cmd, password)
	if err != nil {
		return func() tea.Msg { return SSHExitedMsg{Err: err} }
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		cleanup()
		return SSHExitedMsg{Err: err}
	})
}

// openTunnelCmd starts a background SSH port-forwarding tunnel.
func openTunnelCmd(mgr tunnel.Manager, hostName string, t sshconfig.Tunnel, password string) tea.Cmd {
	return func() tea.Msg {
		err := mgr.Open(hostName, t, password)
		return openTunnelResultMsg{TunnelID: t.ID(hostName), Err: err}
	}
}

// closeTunnelCmd terminates a running tunnel process.
func closeTunnelCmd(mgr tunnel.Manager, tunnelID string) tea.Cmd {
	return func() tea.Msg {
		err := mgr.Close(tunnelID)
		return closeTunnelResultMsg{TunnelID: tunnelID, Err: err}
	}
}
