// Package tunnels implements the tunnel list page, used both for the tunnels
// of a single host and for the global list of open tunnels.
package tunnels

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/key"
	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/system"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/list"
	"github.com/highfredo/ssx/internal/ui/modal"
)

// TunnelPage lists tunnels and lets the user open, close or browse them.
type TunnelPage struct {
	list    *list.Model
	host    *ssh.HostConfig // nil on the "open tunnels" page
	hosts   ssh.Hosts       // resolves tunnel owners on the "open tunnels" page
	tunnels []ssh.Tunnel
	keys    *base.AppKeybindings
}

// tunnelStateChangedMsg carries fresh statuses for one or more tunnels.
type tunnelStateChangedMsg struct {
	reports []ssh.TunnelReport
}

// authFailedMsg reports that ssh rejected the credentials while opening a
// tunnel; the page asks for a password and retries.
type authFailedMsg struct {
	host   *ssh.HostConfig
	tunnel ssh.Tunnel
	status ssh.TunnelStatus
}

// NewForHost creates the page listing the tunnels of host, including a
// virtual "quick tunnel" item built from the search query.
func NewForHost(host *ssh.HostConfig, keys *base.AppKeybindings) *TunnelPage {
	m := newPage("Tunnels for "+host.Name, host.Tunnels, keys, false)
	m.host = host
	m.list.SetEmptyText("No tunnels for " + host.Name + "  ·  type port[:host[:port]] to open a quick tunnel")
	m.list.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			m.keys.Tunnels.Toggle, m.keys.Tunnels.Browser,
			m.keys.Hosts.CopyKey, m.keys.Hosts.FTP, m.keys.Hosts.Info, m.keys.Hosts.EditConfig,
			m.keys.Navigation.Back,
		}
	}
	m.list.SetVirtualItemBuilder(func(query string) blist.Item {
		tun := ssh.ParseTunnel(host.Name, query)
		if tun == nil || m.hasTunnelOnPort(tun.LocalPort) {
			return nil
		}
		return item{tunnel: *tun, status: ssh.TunnelStatus{State: ssh.TunnelClosed}, quick: true}
	})
	return m
}

// NewOpen creates the page listing every open tunnel across hosts.
func NewOpen(hosts ssh.Hosts, keys *base.AppKeybindings) *TunnelPage {
	m := newPage("Open Tunnels", nil, keys, true)
	m.hosts = hosts
	m.list.SetEmptyText("No open tunnels")
	return m
}

func newPage(title string, tunnels []ssh.Tunnel, keys *base.AppKeybindings, showHost bool) *TunnelPage {
	items := make([]blist.Item, len(tunnels))
	for i, t := range tunnels {
		items[i] = item{tunnel: t, status: ssh.TunnelStatus{State: ssh.TunnelUnknown}}
	}

	l := list.New(items, newDelegate(showHost), keys.Navigation)
	l.Title = title
	l.SetStatusFunc(tunnelStatus)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Tunnels.Toggle, keys.Tunnels.Browser, keys.Navigation.Back}
	}
	l.AdditionalFullHelpKeys = l.AdditionalShortHelpKeys

	return &TunnelPage{list: l, tunnels: tunnels, keys: keys}
}

// tunnelStatus summarises the visible tunnels by state.
func tunnelStatus(visible []blist.Item, total int, query string) string {
	var open, inUse, opening int
	for _, li := range visible {
		it, ok := li.(item)
		if !ok || it.quick {
			continue
		}
		switch it.status.State {
		case ssh.TunnelOpened:
			open++
		case ssh.TunnelPortInUse:
			inUse++
		case ssh.TunnelOpening:
			opening++
		}
	}

	noun := "tunnels"
	if total == 1 {
		noun = "tunnel"
	}
	parts := []string{fmt.Sprintf("%d %s", total, noun)}
	if query != "" {
		parts[0] = fmt.Sprintf("%d of %d %s", len(visible), total, noun)
	}
	if open > 0 {
		parts = append(parts, styles.StateOpenStyle.Render(fmt.Sprintf("%d open", open)))
	}
	if opening > 0 {
		parts = append(parts, styles.StateOpeningStyle.Render(fmt.Sprintf("%d opening", opening)))
	}
	if inUse > 0 {
		parts = append(parts, styles.StateInUseStyle.Render(fmt.Sprintf("%d in use", inUse)))
	}
	return strings.Join(parts, "  ·  ")
}

// Init refreshes the status of every tunnel in the background.
func (m *TunnelPage) Init() tea.Cmd {
	host, hosts := m.host, m.hosts
	return func() tea.Msg {
		if host != nil {
			return tunnelStateChangedMsg{reports: ssh.HostTunnels(host)}
		}
		return tunnelStateChangedMsg{reports: ssh.OpenTunnels(hosts)}
	}
}

// Update implements base.Component.
func (m *TunnelPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if cmd, handled := m.handleKey(msg); handled {
			return m, cmd
		}
	case tunnelStateChangedMsg:
		return m, m.applyReports(msg.reports)
	case authFailedMsg:
		return m, tea.Batch(
			m.applyReport(ssh.TunnelReport{Tunnel: msg.tunnel, Status: msg.status}),
			askPassword(msg.host, msg.tunnel),
		)
	}
	return m, m.list.Update(msg)
}

// handleKey resolves action keys against the selected item. It reports false
// when the key should fall through to the list (navigation, search).
func (m *TunnelPage) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	it, _ := m.list.SelectedItem().(item)

	switch {
	case key.Matches(msg, m.keys.Navigation.Back):
		if m.list.SearchValue() != "" {
			m.list.ClearFilter()
			return nil, true
		}
		return base.Emit(base.OpenHostPageMsg{}), true
	case key.Matches(msg, m.keys.Tunnels.Toggle):
		return m.toggleTunnel(it.tunnel, it.status), true
	case key.Matches(msg, m.keys.Tunnels.Browser):
		return openInBrowser(it.tunnel), true
	}

	if h := m.host; h != nil {
		switch {
		case key.Matches(msg, m.keys.Hosts.CopyKey):
			return base.Emit(base.CopySSHKeyMsg{HostConfig: h}), true
		case key.Matches(msg, m.keys.Hosts.FTP):
			return base.Emit(base.OpenFTPMsg{HostConfig: h}), true
		case key.Matches(msg, m.keys.Hosts.Info):
			return base.Emit(base.OpenInfoPageMsg{HostConfig: h}), true
		case key.Matches(msg, m.keys.Hosts.EditConfig):
			return base.Emit(base.EditSSHConfigMsg{HostConfig: h}), true
		}
	}
	return nil, false
}

// applyReports updates list items with new statuses, promoting the virtual
// quick-tunnel item to a permanent one once it is opened and appending
// tunnels discovered at runtime.
func (m *TunnelPage) applyReports(reports []ssh.TunnelReport) tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(reports))
	for _, r := range reports {
		cmds = append(cmds, m.applyReport(r))
	}
	return tea.Batch(cmds...)
}

func (m *TunnelPage) applyReport(r ssh.TunnelReport) tea.Cmd {
	id := r.Tunnel.ID()

	if vItem, vIdx := m.list.VirtualItem(); vItem != nil {
		if virtual, ok := vItem.(item); ok && virtual.tunnel.ID() == id {
			updated := item{tunnel: virtual.tunnel, status: r.Status}
			if r.Status.State != ssh.TunnelOpened {
				return m.list.SetItem(vIdx, updated)
			}
			m.rememberTunnel(virtual.tunnel)
			return m.list.PromoteVirtualItem(updated)
		}
	}

	for i, existing := range m.tunnels {
		if existing.ID() == id {
			return m.list.SetItem(i, item{tunnel: existing, status: r.Status})
		}
	}

	m.tunnels = append(m.tunnels, r.Tunnel)
	return m.list.InsertItem(len(m.tunnels)-1, item{tunnel: r.Tunnel, status: r.Status})
}

// rememberTunnel keeps a quick tunnel for the rest of the session, both in
// this page and in the host so other pages see it too.
func (m *TunnelPage) rememberTunnel(t ssh.Tunnel) {
	if !m.hasTunnelOnPort(t.LocalPort) {
		m.tunnels = append(m.tunnels, t)
	}
	if m.host != nil {
		m.host.Tunnels = append(m.host.Tunnels, t)
	}
}

func (m *TunnelPage) hasTunnelOnPort(localPort string) bool {
	for _, t := range m.tunnels {
		if t.LocalPort == localPort {
			return true
		}
	}
	return false
}

// hostFor resolves the host a tunnel goes through.
func (m *TunnelPage) hostFor(t ssh.Tunnel) *ssh.HostConfig {
	if m.host != nil {
		return m.host
	}
	return m.hosts.Get(t.Host)
}

// SetSize implements base.Component.
func (m *TunnelPage) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

// View implements base.Component.
func (m *TunnelPage) View() string {
	return m.list.View()
}

// toggleTunnel opens a closed tunnel, closes an open one, or offers to free
// the port when another process holds it.
func (m *TunnelPage) toggleTunnel(tunnel ssh.Tunnel, status ssh.TunnelStatus) tea.Cmd {
	host := m.hostFor(tunnel)
	switch status.State {
	case ssh.TunnelClosed, ssh.TunnelUnknown:
		return openTunnel(host, tunnel, nil)
	case ssh.TunnelOpened:
		return closeTunnel(host, tunnel)
	case ssh.TunnelPortInUse:
		owner := status.Owner
		if owner == "" {
			owner = "another process"
		}
		question := fmt.Sprintf("Port %s is in use by %s. Kill it and open tunnel?", tunnel.LocalPort, owner)
		return modal.OpenDangerConfirm("Port in use", question, func(confirmed bool) tea.Cmd {
			if !confirmed {
				return nil
			}
			return openTunnel(host, tunnel, func() error { return ssh.KillPortOwner(tunnel.LocalPort) })
		})
	}
	return nil
}

// openTunnel marks the tunnel as opening, runs the optional prepare step and
// then spawns the tunnel, reporting the resulting status.
func openTunnel(host *ssh.HostConfig, tunnel ssh.Tunnel, prepare func() error) tea.Cmd {
	return tea.Sequence(
		base.Emit(singleReport(tunnel, ssh.TunnelStatus{State: ssh.TunnelOpening})),
		func() tea.Msg {
			if prepare != nil {
				if err := prepare(); err != nil {
					return statusAfterError(host, tunnel, err)
				}
			}
			st, err := ssh.OpenTunnel(host, tunnel)
			if errors.Is(err, ssh.ErrAuthFailed) {
				return authFailedMsg{host: host, tunnel: tunnel, status: st}
			}
			return singleReport(tunnel, st)
		},
	)
}

// askPassword opens the password modal and retries the tunnel with a copy of
// the host that authenticates with the entered password.
func askPassword(host *ssh.HostConfig, tunnel ssh.Tunnel) tea.Cmd {
	return modal.OpenPassword(
		"Authentication failed",
		fmt.Sprintf("Enter the password for %s.", host.Target()),
		func(pw string) tea.Cmd { return openTunnel(host.WithPassword(pw), tunnel, nil) },
	)
}

func closeTunnel(host *ssh.HostConfig, tunnel ssh.Tunnel) tea.Cmd {
	return func() tea.Msg {
		return statusAfterError(host, tunnel, ssh.CloseTunnel(tunnel))
	}
}

// statusAfterError re-reads the tunnel status and attaches err to it.
func statusAfterError(host *ssh.HostConfig, tunnel ssh.Tunnel, err error) tea.Msg {
	st, _ := ssh.GetTunnelStatus(host, tunnel)
	st.Err = err
	return singleReport(tunnel, st)
}

func singleReport(t ssh.Tunnel, st ssh.TunnelStatus) tunnelStateChangedMsg {
	return tunnelStateChangedMsg{reports: []ssh.TunnelReport{{Tunnel: t, Status: st}}}
}

// openInBrowser opens http(s)://localhost:<port> with the system opener.
func openInBrowser(tunnel ssh.Tunnel) tea.Cmd {
	return func() tea.Msg {
		scheme := "http"
		if tunnel.LocalPort == "443" || tunnel.LocalPort == "8443" {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://localhost:%s", scheme, tunnel.LocalPort)
		if err := system.Open(url); err != nil {
			slog.Error("cannot open browser", "url", url, "err", err)
			return modal.OpenAlert("Cannot open browser", err.Error(), true)()
		}
		return nil
	}
}
