package tunnels

import (
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/key"
	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/opener"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/list"
	"github.com/highfredo/ssx/internal/ui/modal"
)

var keys = struct {
	Toggle  key.Binding
	Browser key.Binding
	Back    key.Binding
}{
	Toggle:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "toggle tunnel")),
	Browser: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "open in browser")),
	Back:    key.NewBinding(key.WithKeys("esc", "tab"), key.WithHelp("esc/tab", "back")),
}

type TunnelPage struct {
	list    *list.Model
	tunnels []ssh.Tunnel
	manager *ssh.TunnelManager
}

func New(title string, tunnels []ssh.Tunnel, manager *ssh.TunnelManager) *TunnelPage {
	items := make([]blist.Item, len(tunnels))
	for i, t := range tunnels {
		tunnelStatus, portStatus := manager.Status(t)
		items[i] = item{tunnel: t, tunnelStatus: tunnelStatus, portStatus: portStatus}
	}

	d := list.NewDefaultDelegate()

	l := list.New(items, d, 0, 0)
	l.Title = title
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Toggle, keys.Browser, keys.Back}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Toggle, keys.Browser, keys.Back}
	}

	return &TunnelPage{list: l, tunnels: tunnels, manager: manager}
}

func (m *TunnelPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		// Action keys — resolved against the currently selected item.
		item, _ := m.list.SelectedItem().(item)
		switch {
		case key.Matches(keyMsg, keys.Back):
			return m, func() tea.Msg { return base.OpenHostPageMsg{} }
		case key.Matches(keyMsg, keys.Toggle):
			return m, m.ToggleTunnel(item.tunnel, item.tunnelStatus, item.portStatus)
		case key.Matches(keyMsg, keys.Browser):
			return m, m.OpenInBrowser(item.tunnel)
		}
	}

	switch msg.(type) {
	case ssh.TunnelStateChangedMsg:
		for i, t := range m.tunnels {
			tunnelStatus, portStatus := m.manager.Status(t)
			m.list.SetItem(i, item{tunnel: t, tunnelStatus: tunnelStatus, portStatus: portStatus})
		}
		return m, nil
	}

	return m, m.list.Update(msg)
}

func (m *TunnelPage) SetSize(w, h int) {
	slog.Info("resizing tunnel list", "width", w, "height", h)
	m.list.SetSize(w, h)
}

func (m *TunnelPage) View() string {
	return m.list.View()
}

func (m *TunnelPage) ToggleTunnel(tunnel ssh.Tunnel, tunnelStatus ssh.TunnelStatus, portStatus ssh.PortStatus) tea.Cmd {
	if tunnelStatus.State == ssh.TunnelClosed && portStatus.State == ssh.PortOpened {
		return modal.OpenConfirm("Port in use", "placeholder", func(confirmed bool) tea.Cmd {
			if !confirmed {
				return nil
			}
			return func() tea.Msg {
				ssh.KillPortOwner(tunnel.LocalPort)
				m.manager.Open(tunnel)
				return nil
			}
		})
	} else if tunnelStatus.State == ssh.TunnelClosed {
		return func() tea.Msg {
			m.manager.Open(tunnel)
			return nil
		}
	} else if tunnelStatus.State == ssh.TunnelOpened {
		return func() tea.Msg {
			m.manager.Close(tunnel)
			return nil
		}
	}

	return nil
}

func (m *TunnelPage) OpenInBrowser(tunnel ssh.Tunnel) tea.Cmd {
	return func() tea.Msg {
		scheme := "http"
		if tunnel.LocalPort == "443" || tunnel.LocalPort == "8443" {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://localhost:%s", scheme, tunnel.LocalPort)

		return opener.Open("/mnt/c/Windows/explorer.exe", url)
	}
}
