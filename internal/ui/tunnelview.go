package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/highfredo/ssx/internal/sshconfig"
	"github.com/highfredo/ssx/internal/tunnel"
)

// ── TunnelView model ──────────────────────────────────────────────────────────

// TunnelView is the Bubble Tea model for the tunnel management screen.
// It displays every configured tunnel for a selected host with its live state
// (OPEN / CLOSED) and allows the user to open or close individual tunnels.
type TunnelView struct {
	host   *sshconfig.Host
	mgr    tunnel.Manager
	cursor int
	width  int
	height int
}

// NewTunnelView creates a TunnelView for the given host and manager.
func NewTunnelView(host *sshconfig.Host, mgr tunnel.Manager, w, h int) TunnelView {
	return TunnelView{
		host:   host,
		mgr:    mgr,
		cursor: 0,
		width:  w,
		height: h,
	}
}

// SetSize updates the view dimensions on terminal resize.
func (tv *TunnelView) SetSize(w, h int) {
	tv.width = w
	tv.height = h
}

// Update handles keyboard input for the tunnel screen.
func (tv TunnelView) Update(msg tea.Msg) (TunnelView, tea.Cmd) {
	tunnels := tv.host.Tunnels

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		// Navigation
		case "up", "k":
			if tv.cursor > 0 {
				tv.cursor--
			}

		case "down", "j":
			if tv.cursor < len(tunnels)-1 {
				tv.cursor++
			}

		// Open selected tunnel
		case "o", "O":
			if len(tunnels) > 0 {
				t := tunnels[tv.cursor]
				id := t.ID(tv.host.Name)
				if tv.mgr.State(id) == tunnel.StateOpen {
					// Already open — no-op, but return a result msg so the
					// status bar explains it.
					return tv, func() tea.Msg {
						return openTunnelResultMsg{
							TunnelID: id,
							Err:      fmt.Errorf("tunnel is already open"),
						}
					}
				}
				return tv, func() tea.Msg {
					return requestOpenTunnelMsg{hostName: tv.host.Name, tunnel: t}
				}
			}

		// Toggle selected tunnel
		case "x", "X":
			if len(tunnels) > 0 {
				t := tunnels[tv.cursor]
				id := t.ID(tv.host.Name)
				if tv.mgr.State(id) == tunnel.StateClosed {
					return tv, func() tea.Msg {
						return requestOpenTunnelMsg{hostName: tv.host.Name, tunnel: t}
					}
				}
				return tv, closeTunnelCmd(tv.mgr, id)
			}

		// Back to host list
		case "q", "Q", "esc":
			return tv, func() tea.Msg { return backMsg{} }
		}

	// Re-render on any refresh message (tunnel state changed externally).
	case refreshTunnelViewMsg:
		// No state to update — View() reads live state from the manager.
	}

	return tv, nil
}

// View renders the tunnel management panel.
func (tv TunnelView) View() string {
	var sb strings.Builder

	// ── Title ─────────────────────────────────────────────────────────────────
	title := titleStyle.Render(fmt.Sprintf(" Tunnels › %s ", tv.host.Name))
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// ── Tunnel rows ──────────────────────────────────────────────────────────
	if len(tv.host.Tunnels) == 0 {
		sb.WriteString(helpStyle.Render("  No tunnels configured for this host."))
		sb.WriteString("\n")
	} else {
		for i, t := range tv.host.Tunnels {
			sb.WriteString(tv.renderRow(i, t))
			sb.WriteString("\n")
		}
	}

	// ── Help bar ─────────────────────────────────────────────────────────────
	sb.WriteString("\n")
	help := helpStyle.Render("[x] toggle open/close  [↑↓ / jk] navigate  [esc] back")
	sb.WriteString(help)

	return sb.String()
}

// renderRow renders a single tunnel entry with type badge, spec, and state.
func (tv TunnelView) renderRow(idx int, t sshconfig.Tunnel) string {
	id := t.ID(tv.host.Name)
	state := tv.mgr.State(id)
	isSelected := idx == tv.cursor

	// Cursor indicator
	cursor := "   "
	if isSelected {
		cursor = cursorStyle.Render(" ❯ ")
	}

	// Type badge — fixed 8-char visual width, colour-coded
	typeLabel := tunnelTypeStyle(string(t.Type)).
		Render(fmt.Sprintf("%-7s", strings.ToUpper(string(t.Type))))

	// Forwarding specification — left-padded to 40 visible chars
	specText := t.DisplaySpec()
	if t.Description != "" {
		specText = fmt.Sprintf("%s  # %s", specText, t.Description)
	}
	spec := fmt.Sprintf("%-40s", specText)

	// State indicator
	var stateLabel string
	if state == tunnel.StateOpen {
		stateLabel = stateOpenStyle.Render("● OPEN  ")
	} else {
		stateLabel = stateClosedStyle.Render("○ CLOSED")
	}

	row := cursor + typeLabel + "  " + spec + "  " + stateLabel

	if isSelected {
		// Apply background highlight; preserve inner colour codes.
		row = tunnelRowSelectedStyle.
			Width(tv.usableWidth()).
			Render(lipgloss.NewStyle().Render(row))
	}

	return row
}

// usableWidth returns a safe column width for the selected row highlight.
func (tv TunnelView) usableWidth() int {
	if tv.width > 4 {
		return tv.width - 4
	}
	return 80
}
