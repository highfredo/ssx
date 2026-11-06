// Package hosts implements the root page listing the configured SSH hosts.
package hosts

import (
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/key"
	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/system"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/list"
	"github.com/highfredo/ssx/internal/ui/modal"
)

// HostPage lists hosts and dispatches host-level actions.
type HostPage struct {
	list *list.Model
	keys base.HostsKeys
	back key.Binding
}

// openShellMsg requests an interactive shell on the given host.
type openShellMsg struct {
	host *ssh.HostConfig
}

// New creates the host page. tagColors provides default chip colours per tag
// name. A virtual "quick connect" item is built when the search query looks
// like a connection string (user@host:port).
func New(hosts ssh.Hosts, tagColors map[string]string, allKeys *base.AppKeybindings) *HostPage {
	keys := allKeys.Hosts
	items := make([]blist.Item, len(hosts))
	for i, h := range hosts {
		items[i] = item{host: h}
	}

	configPath := paths.Shorten(paths.SSHConfigFile())
	l := list.New(items, hostDelegate{DefaultDelegate: list.NewDefaultDelegate(), tagColors: tagColors}, allKeys.Navigation)
	l.Title = "SSX · " + configPath
	l.SetStatusFunc(countStatus("host", "hosts"))
	l.SetEmptyText("No hosts in " + configPath + "  ·  type user@host[:port] to quick connect")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Connect, keys.Tunnels, keys.FTP}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Connect, keys.CopyKey, keys.Tunnels, keys.OpenTunnels,
			keys.FTP, keys.Info, keys.EditConfig, keys.OpenConfig,
			keys.QuickConnect,
		}
	}
	l.SetVirtualItemBuilder(func(query string) blist.Item {
		if con := ssh.ParseConnection(query); con != nil {
			con.Title = "⚡Quick Connect"
			return item{host: con}
		}
		return nil
	})

	return &HostPage{list: l, keys: keys, back: allKeys.Navigation.Back}
}

// countStatus builds a "N things" / "N of M things" counter.
func countStatus(singular, plural string) list.StatusFunc {
	return func(visible []blist.Item, total int, query string) string {
		noun := plural
		if total == 1 {
			noun = singular
		}
		if query != "" {
			return fmt.Sprintf("%d of %d %s", len(visible), total, noun)
		}
		return fmt.Sprintf("%d %s", total, noun)
	}
}

// Update implements base.Component.
func (m *HostPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if cmd, handled := m.handleKey(msg); handled {
			return m, cmd
		}
	case openShellMsg:
		return m, openShell(msg.host)
	}
	return m, m.list.Update(msg)
}

// handleKey resolves action keys against the selected item. It reports false
// when the key should fall through to the list (navigation, search).
func (m *HostPage) handleKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	it, _ := m.list.SelectedItem().(item)
	h := it.host

	switch {
	case key.Matches(msg, m.keys.FTP):
		return base.Emit(base.OpenFTPMsg{HostConfig: h}), true
	case key.Matches(msg, m.keys.OpenTunnels):
		return base.Emit(base.OpenTunnelsOpenPageMsg{}), true
	case key.Matches(msg, m.keys.Tunnels):
		return base.Emit(base.OpenTunnelForHostPageMsg{HostConfig: h}), true
	case key.Matches(msg, m.keys.Connect):
		return base.Emit(openShellMsg{host: h}), true
	case key.Matches(msg, m.keys.QuickConnect):
		vItem, _ := m.list.VirtualItem()
		quick, ok := vItem.(item)
		if !ok || quick.host == nil {
			return nil, true
		}
		target := quick.host
		return base.Emit(openShellMsg{host: target}), true
	case key.Matches(msg, m.keys.CopyKey):
		return base.Emit(base.CopySSHKeyMsg{HostConfig: h}), true
	case key.Matches(msg, m.keys.Info):
		return base.Emit(base.OpenInfoPageMsg{HostConfig: h}), true
	case key.Matches(msg, m.keys.EditConfig):
		return base.Emit(base.EditSSHConfigMsg{HostConfig: h}), true
	case key.Matches(msg, m.keys.OpenConfig):
		return openAppConfig(), true
	// Checked last: Back shares "tab" with Tunnels, so actions take priority.
	case key.Matches(msg, m.back):
		if m.list.SearchValue() != "" {
			m.list.ClearFilter()
		}
		return nil, true
	}
	return nil, false
}

// SetSize implements base.Component.
func (m *HostPage) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

// View implements base.Component.
func (m *HostPage) View() string {
	return m.list.View()
}

// openShell hands the terminal over to an interactive ssh session.
func openShell(h *ssh.HostConfig) tea.Cmd {
	slog.Info("opening shell", "name", h.Name)
	return tea.ExecProcess(ssh.Shell(h), func(err error) tea.Msg {
		if err != nil {
			slog.Error("error opening shell", "err", err)
			return tea.Quit()
		}
		return nil
	})
}

// openAppConfig opens the ssx config file with the system default handler.
func openAppConfig() tea.Cmd {
	cfgFile := paths.AppConfigFile()
	return func() tea.Msg {
		if err := system.Open(cfgFile); err != nil {
			return modal.OpenAlert("Cannot open config", err.Error(), true)()
		}
		return nil
	}
}
