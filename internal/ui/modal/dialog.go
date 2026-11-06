// Package modal provides the dialogs rendered on top of the active page:
// alerts, confirmations and a busy spinner.
package modal

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

const (
	dialogWidth   = 60
	dialogPadding = 2
	dialogBorder  = 1
	// contentWidth is the inner width: lipgloss widths include border and padding.
	contentWidth = dialogWidth - 2*(dialogPadding+dialogBorder)
)

var dialogCenterStyle = lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)

// renderDialog draws a bordered dialog with a highlighted title, a message
// body, an optional centred row of buttons and a key hint, all in the accent colour.
func renderDialog(accent color.Color, title, body, buttons, hint string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.ColorText).
		Background(accent).
		Padding(0, 1).
		Width(contentWidth)

	rows := []string{
		titleStyle.Render(title),
		"",
		lipgloss.NewStyle().Foreground(styles.ColorText).Width(contentWidth).Render(body),
		"",
	}
	if buttons != "" {
		rows = append(rows, dialogCenterStyle.Render(buttons), "")
	}
	// Resolved at render time so a user theme applied after init is honoured.
	rows = append(rows, dialogCenterStyle.Foreground(styles.ColorMuted).Render(hint))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Padding(1, dialogPadding).
		Width(dialogWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// renderButton draws a button, highlighted with accent when active.
func renderButton(label string, active bool, accent color.Color) string {
	st := lipgloss.NewStyle().Padding(0, 2).Bold(true)
	if active {
		st = st.Background(accent).Foreground(styles.ColorText)
	} else {
		st = st.Background(styles.ColorBorder).Foreground(styles.ColorMuted)
	}
	return st.Render(label)
}

var closeModal = base.Emit(base.CloseModalMsg{})
