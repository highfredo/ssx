// Package ui contains all Bubble Tea TUI models, views, and styles for ssx.
package ui

import "charm.land/lipgloss/v2"

// ── Color palette ────────────────────────────────────────────────────────────

var (
	colorPrimary   = lipgloss.Color("#7C3AED") // Purple  — accent / title bg
	colorSecondary = lipgloss.Color("#4F46E5") // Indigo  — selected item bg
	colorHighlight = lipgloss.Color("#1E1B4B") // Dark indigo — selected row bg
	colorSuccess   = lipgloss.Color("#10B981") // Green   — OPEN state
	colorMuted     = lipgloss.Color("#6B7280") // Gray    — CLOSED state / help
	colorDanger    = lipgloss.Color("#EF4444") // Red     — errors
	colorBorder    = lipgloss.Color("#374151") // Dark gray — borders

	colorTypeLocal   = lipgloss.Color("#60A5FA") // Blue
	colorTypeRemote  = lipgloss.Color("#F97316") // Orange
	colorTypeDynamic = lipgloss.Color("#A78BFA") // Violet
)

// ── Shared styles ─────────────────────────────────────────────────────────────

var (
	// titleStyle is used for the panel title bars.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(colorPrimary).
			Padding(0, 1)

	// helpStyle renders the bottom key-hint bar.
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	// errorStyle renders inline error/status messages.
	errorStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Padding(0, 1)

	// statusOKStyle renders success status messages.
	statusOKStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Padding(0, 1)

	// stateOpenStyle highlights OPEN tunnel state.
	stateOpenStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSuccess)

	// stateClosedStyle renders CLOSED tunnel state.
	stateClosedStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	// tunnelRowSelectedStyle highlights the focused tunnel row.
	tunnelRowSelectedStyle = lipgloss.NewStyle().
				Background(colorHighlight).
				Foreground(lipgloss.Color("#E0E7FF"))

	// cursorStyle renders the ❯ indicator on selected rows.
	cursorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	// panelStyle draws a rounded border around the tunnel view.
	panelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	// tunnelTypeLabelStyle is the base for type badge — colour applied per type.
	tunnelTypeLabelBase = lipgloss.NewStyle().Bold(true)

	// tagChipStyle renders host tags as small pill-shaped badges.
	tagChipStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#312E81")).
			Foreground(lipgloss.Color("#C4B5FD")).
			Padding(0, 1).
			Bold(true)
)

// TunnelTypeStyle returns the colour-appropriate style for a tunnel type label.
func tunnelTypeStyle(typ string) lipgloss.Style {
	switch typ {
	case "local":
		return tunnelTypeLabelBase.Foreground(colorTypeLocal)
	case "remote":
		return tunnelTypeLabelBase.Foreground(colorTypeRemote)
	default: // dynamic
		return tunnelTypeLabelBase.Foreground(colorTypeDynamic)
	}
}
