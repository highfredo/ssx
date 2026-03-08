package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/colors"
)

// ── Color palette ────────────────────────────────────────────────────────────

var theme = appconfig.Get().Theme

var (
	ColorPrimary = getColorOrDefault(theme.Primary, "#7C3AED") // Purple     — accent / title bg
	// ColorSecondary = lipgloss.Color("#4F46E5") // Indigo  — selected item bg
	ColorHighlight = getColorOrDefault(theme.Highlight, "#1E1B4B") // Dark indigo — selected row bg
	ColorSuccess   = getColorOrDefault(theme.Success, "#10B981")   // Green      — OPEN state
	ColorMuted     = getColorOrDefault(theme.Muted, "#6B7280")     // Gray       — CLOSED state / help
	ColorDanger    = getColorOrDefault(theme.Danger, "#EF4444")    // Red        — errors
	ColorBorder    = getColorOrDefault(theme.Border, "#374151")    // Dark gray  — borders

	ColorDefaultTag = getColorOrDefault(theme.DefaultTag, "#312E81")
)

func getColorOrDefault(a string, def string) color.Color {
	if a != "" {
		return lipgloss.Color(a)
	}
	return lipgloss.Color(def)
}

// ── Shared styles ─────────────────────────────────────────────────────────────

var (
	// TitleStyle is used for panel title bars.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(ColorPrimary).
			Padding(0, 1)

	// FilterBarStyle is used for filter input bars across all pages.
	FilterBarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// StateOpenStyle highlights OPEN tunnel state.
	StateOpenStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	// StateClosedStyle renders CLOSED tunnel state.
	StateClosedStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// PanelStyle draws a rounded border around panels.
	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// TagChipStyle renders host tags as small pill-shaped badges.
	TagChipStyle = lipgloss.NewStyle().
			Background(ColorDefaultTag).
			Foreground(colors.TintedForeground(colors.Hex(ColorDefaultTag))).
			Padding(0, 1).
			Bold(true)
)
