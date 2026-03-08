package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/highfredo/ssx/internal/sshconfig"
)

type openedTunnelRow struct {
	TunnelID string
	Host     string
	Tunnel   sshconfig.Tunnel
	Spec     string
	State    string
}

// OpenedTunnelsView renders currently opened tunnels.
type OpenedTunnelsView struct {
	rows   []openedTunnelRow
	cursor int
	width  int
	height int
}

func NewOpenedTunnelsView(rows []openedTunnelRow, w, h int) OpenedTunnelsView {
	return OpenedTunnelsView{
		rows:   rows,
		width:  w,
		height: h,
	}
}

func (v *OpenedTunnelsView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *OpenedTunnelsView) SetRows(rows []openedTunnelRow) {
	v.rows = rows
	if len(v.rows) == 0 {
		v.cursor = 0
		return
	}
	if v.cursor >= len(v.rows) {
		v.cursor = len(v.rows) - 1
	}
}

// RefreshRows updates open/closed states while keeping rows already shown.
func (v *OpenedTunnelsView) RefreshRows(openRows []openedTunnelRow) {
	if len(v.rows) == 0 {
		v.SetRows(openRows)
		return
	}

	openByID := make(map[string]openedTunnelRow, len(openRows))
	for _, row := range openRows {
		openByID[row.TunnelID] = row
	}

	for i := range v.rows {
		if openRow, ok := openByID[v.rows[i].TunnelID]; ok {
			v.rows[i].State = openRow.State
			v.rows[i].Spec = openRow.Spec
			v.rows[i].Tunnel = openRow.Tunnel
			delete(openByID, v.rows[i].TunnelID)
		} else {
			v.rows[i].State = "CLOSED"
		}
	}

	// Include newly opened tunnels that were not visible yet.
	for _, row := range openByID {
		v.rows = append(v.rows, row)
	}
	if v.cursor >= len(v.rows) {
		v.cursor = max(0, len(v.rows)-1)
	}
}

func (v OpenedTunnelsView) Update(msg tea.Msg) (OpenedTunnelsView, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.rows)-1 {
				v.cursor++
			}
		case "x", "X", "enter":
			if len(v.rows) > 0 && v.cursor >= 0 && v.cursor < len(v.rows) {
				row := v.rows[v.cursor]
				if row.State == "OPEN" {
					return v, func() tea.Msg { return requestCloseTunnelMsg{tunnelID: row.TunnelID} }
				}
				return v, func() tea.Msg {
					return requestOpenTunnelMsg{hostName: row.Host, tunnel: row.Tunnel}
				}
			}
		case "q", "Q", "esc", "o", "O":
			return v, func() tea.Msg { return backMsg{} }
		}
		return v, nil
	}
	if mouse, ok := msg.(tea.MouseMsg); ok {
		switch mouse.Button {
		case tea.MouseButtonWheelUp:
			if v.cursor > 0 {
				v.cursor--
			}
		case tea.MouseButtonWheelDown:
			if v.cursor < len(v.rows)-1 {
				v.cursor++
			}
		case tea.MouseButtonLeft:
			row := mouse.Y - 2
			if row >= 0 && row < len(v.rows) {
				if row == v.cursor {
					selected := v.rows[row]
					if selected.State == "OPEN" {
						return v, func() tea.Msg { return requestCloseTunnelMsg{tunnelID: selected.TunnelID} }
					}
					return v, func() tea.Msg {
						return requestOpenTunnelMsg{hostName: selected.Host, tunnel: selected.Tunnel}
					}
				}
				v.cursor = row
			}
		}
		return v, nil
	}
	return v, nil
}

func (v OpenedTunnelsView) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(" Opened Tunnels "))
	sb.WriteString("\n\n")
	if len(v.rows) == 0 {
		sb.WriteString(helpStyle.Render("  No tunnels are currently open."))
		sb.WriteString("\n\n")
		sb.WriteString(helpStyle.Render("[o/esc/q] back"))
		return sb.String()
	}

	for i, row := range v.rows {
		cursor := "   "
		if i == v.cursor {
			cursor = cursorStyle.Render(" ❯ ")
		}
		line := fmt.Sprintf("%s%-20s %-48s %s", cursor, row.Host, row.Spec, row.State)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("[enter/x] toggle selected  [↑↓ / jk] navigate  [o/esc/q] back"))
	return sb.String()
}
