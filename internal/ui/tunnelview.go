package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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
	host         *sshconfig.Host
	mgr          tunnel.Manager
	cursor       int
	width        int
	height       int
	filterInput  textinput.Model
	filterActive bool
	portOwners   map[string]*tunnel.PortOwner
}

// NewTunnelView creates a TunnelView for the given host and manager.
func NewTunnelView(host *sshconfig.Host, mgr tunnel.Manager, w, h int) TunnelView {
	filter := textinput.New()
	filter.Prompt = "Filter: "
	filter.Placeholder = "type to filter tunnels"
	filter.CharLimit = 128
	return TunnelView{
		host:        host,
		mgr:         mgr,
		cursor:      0,
		width:       w,
		height:      h,
		filterInput: filter,
		portOwners:  make(map[string]*tunnel.PortOwner),
	}
}

// SetSize updates the view dimensions on terminal resize.
func (tv *TunnelView) SetSize(w, h int) {
	tv.width = w
	tv.height = h
}

// SetPortOwners stores the latest local port ownership snapshot.
func (tv *TunnelView) SetPortOwners(owners map[string]*tunnel.PortOwner) {
	tv.portOwners = owners
}

// Update handles keyboard input for the tunnel screen.
func (tv TunnelView) Update(msg tea.Msg) (TunnelView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if tv.filterActive {
			switch msg.String() {
			case "enter":
				tv.filterActive = false
				tv.filterInput.Blur()
				return tv, nil
			case "esc":
				tv.filterActive = false
				tv.filterInput.Blur()
				return tv, nil
			}
			var cmd tea.Cmd
			tv.filterInput, cmd = tv.filterInput.Update(msg)
			tv.clampCursor(len(tv.filteredTunnelIndexes()))
			return tv, cmd
		}

		filtered := tv.filteredTunnelIndexes()
		switch msg.String() {
		case "/":
			tv.filterActive = true
			tv.filterInput.Focus()
			return tv, nil

		// Navigation
		case "up", "k":
			if tv.cursor > 0 {
				tv.cursor--
			}

		case "down", "j":
			if tv.cursor < len(filtered)-1 {
				tv.cursor++
			}

		// Open selected tunnel
		case "o", "O":
			if t, ok := tv.selectedTunnel(filtered); ok {
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
			if t, ok := tv.selectedTunnel(filtered); ok {
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
	sb.WriteString("\n")
	if tv.filterActive {
		sb.WriteString(tv.filterInput.View())
	} else {
		query := tv.filterInput.Value()
		if query == "" {
			query = "(none)"
		}
		sb.WriteString(helpStyle.Render(fmt.Sprintf("Filter: %s   [/ edit filter]", query)))
	}
	sb.WriteString("\n\n")

	// ── Tunnel rows ──────────────────────────────────────────────────────────
	if len(tv.host.Tunnels) == 0 {
		sb.WriteString(helpStyle.Render("  No tunnels configured for this host."))
		sb.WriteString("\n")
	} else {
		filtered := tv.filteredTunnelIndexes()
		if len(filtered) == 0 {
			sb.WriteString(helpStyle.Render("  No tunnels match the current filter."))
			sb.WriteString("\n")
		}
		for i, idx := range filtered {
			t := tv.host.Tunnels[idx]
			sb.WriteString(tv.renderRow(i, t))
			sb.WriteString("\n")
		}
	}

	// ── Help bar ─────────────────────────────────────────────────────────────
	sb.WriteString("\n")
	help := helpStyle.Render("[/] filter  [x] toggle open/close  [↑↓ / jk] navigate  [esc] back")
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
	if owner := tv.portOwners[id]; state == tunnel.StateClosed && owner != nil {
		row += "  " + errorStyle.Render(fmt.Sprintf("PORT IN USE by %s (pid %d)", owner.Command, owner.PID))
	}

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

func (tv *TunnelView) clampCursor(filteredCount int) {
	if filteredCount <= 0 {
		tv.cursor = 0
		return
	}
	if tv.cursor >= filteredCount {
		tv.cursor = filteredCount - 1
	}
	if tv.cursor < 0 {
		tv.cursor = 0
	}
}

func (tv TunnelView) filteredTunnelIndexes() []int {
	var out []int
	query := strings.ToLower(strings.TrimSpace(tv.filterInput.Value()))
	for i, t := range tv.host.Tunnels {
		if query == "" || strings.Contains(strings.ToLower(tv.tunnelSearchText(t)), query) {
			out = append(out, i)
		}
	}
	return out
}

func (tv TunnelView) selectedTunnel(filtered []int) (sshconfig.Tunnel, bool) {
	if len(filtered) == 0 || tv.cursor >= len(filtered) || tv.cursor < 0 {
		return sshconfig.Tunnel{}, false
	}
	return tv.host.Tunnels[filtered[tv.cursor]], true
}

func (tv TunnelView) tunnelSearchText(t sshconfig.Tunnel) string {
	return strings.ToLower(strings.Join([]string{
		string(t.Type),
		t.DisplaySpec(),
		t.Description,
	}, " "))
}
