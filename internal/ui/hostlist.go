package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/highfredo/ssx/internal/sshconfig"
)

// ── list.Item implementation ─────────────────────────────────────────────────

// hostItem wraps *sshconfig.Host to satisfy the bubbles/list.Item interface.
type hostItem struct {
	host *sshconfig.Host
}

func (i hostItem) Title() string { return i.host.Name }

func (i hostItem) Description() string {
	var addr string
	if i.host.User != "" {
		addr = fmt.Sprintf("%s@%s", i.host.User, i.host.Hostname)
	} else {
		addr = i.host.Hostname
	}
	if i.host.Port != "22" {
		addr += ":" + i.host.Port
	}
	details := []string{addr}
	if len(i.host.Tags) > 0 {
		details = append(details, "tags: "+strings.Join(i.host.Tags, ", "))
	}
	n := len(i.host.Tunnels)
	if n > 0 {
		details = append(details, fmt.Sprintf("%d tunnel(s) configured", n))
	}
	return strings.Join(details, "   ")
}

func (i hostItem) FilterValue() string {
	parts := []string{i.host.Name, i.host.Hostname, i.host.User}
	parts = append(parts, i.host.Tags...)
	return strings.Join(parts, " ")
}

// ── HostList model ────────────────────────────────────────────────────────────

// HostList is the Bubble Tea model for the SSH host selection screen.
// Navigation (j/k, ↑↓, PgUp/PgDn, /) is handled by the underlying
// bubbles/list component. Custom keys (enter/c, t, q) are intercepted before
// forwarding to the list.
type HostList struct {
	list  list.Model
	hosts []*sshconfig.Host
}

// NewHostList creates a HostList model from a slice of SSH hosts.
func NewHostList(hosts []*sshconfig.Host) HostList {
	items := make([]list.Item, len(hosts))
	for i, h := range hosts {
		items[i] = hostItem{host: h}
	}

	d := list.NewDefaultDelegate()
	// Style the selected item with the accent colour.
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Foreground(lipgloss.Color("#FAFAFA")).
		BorderLeftForeground(colorPrimary)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#C4B5FD")).
		BorderLeftForeground(colorPrimary)

	l := list.New(items, d, 0, 0)
	l.Title = "SSH Manager"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	// Disable built-in quit bindings so the app controls quitting.
	l.DisableQuitKeybindings()

	return HostList{list: l, hosts: hosts}
}

// SetSize resizes the list to fill the available terminal area.
func (h *HostList) SetSize(w, ht int) {
	h.list.SetSize(w, ht)
}

// SelectedHost returns the currently highlighted host, or nil if none.
func (h HostList) SelectedHost() *sshconfig.Host {
	item, ok := h.list.SelectedItem().(hostItem)
	if !ok {
		return nil
	}
	return item.host
}

// Update handles keyboard events and delegates navigation to bubbles/list.
// Custom keys are intercepted only when the list is not in filter mode.
func (h HostList) Update(msg tea.Msg) (HostList, tea.Cmd) {
	if mouse, ok := msg.(tea.MouseMsg); ok && h.list.FilterState() != list.Filtering {
		switch mouse.Button {
		case tea.MouseButtonWheelUp:
			h.list.CursorUp()
			return h, nil
		case tea.MouseButtonWheelDown:
			h.list.CursorDown()
			return h, nil
		case tea.MouseButtonLeft:
			if host := h.SelectedHost(); host != nil {
				return h, func() tea.Msg { return requestConnectMsg{host: host} }
			}
		case tea.MouseButtonRight:
			if host := h.SelectedHost(); host != nil {
				return h, func() tea.Msg { return openTunnelViewMsg{host: host} }
			}
		}
	}

	// Intercept key events when not filtering.
	if key, ok := msg.(tea.KeyMsg); ok && h.list.FilterState() != list.Filtering {
		switch key.String() {
		case "q", "Q":
			return h, tea.Quit

		case "c", "enter":
			if host := h.SelectedHost(); host != nil {
				return h, func() tea.Msg { return requestConnectMsg{host: host} }
			}

		case "t", "T":
			if host := h.SelectedHost(); host != nil {
				return h, func() tea.Msg { return openTunnelViewMsg{host: host} }
			}
		}
	}

	newList, cmd := h.list.Update(msg)
	h.list = newList
	return h, cmd
}

// View renders the host list.
func (h HostList) View() string {
	return h.list.View()
}
