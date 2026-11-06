// Package styles defines the lipgloss palette and shared styles, themed from
// the user configuration.
//
// Everything here is rebuilt by Apply. Callers must read these variables at
// render time (never capture them in package-level vars) so a theme applied
// after init is honoured.
package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/colors"
)

// ── Color palette ────────────────────────────────────────────────────────────

// Palette colours. Defaults are applied at init; Apply overrides them from
// the user theme and rebuilds the derived styles.
var (
	ColorPrimary    color.Color // accent / title background
	ColorSecondary  color.Color // selected description, section titles, keys
	ColorHighlight  color.Color // selected row background
	ColorText       color.Color // main text on accent / selected rows
	ColorSuccess    color.Color // OPEN tunnel state
	ColorWarning    color.Color // transitional state (opening)
	ColorMuted      color.Color // CLOSED tunnel state / help text
	ColorDanger     color.Color // errors / port in use
	ColorBorder     color.Color // panel borders / inactive buttons
	ColorDefaultTag color.Color // tag chips without an explicit colour
)

func init() {
	Apply(appconfig.ThemeConfig{})
}

// Apply sets the palette from theme (empty fields keep the default colour)
// and rebuilds every shared style. Call it once at startup, before any view
// is rendered.
func Apply(theme appconfig.ThemeConfig) {
	ColorPrimary = colorOrDefault(theme.Primary, "#7C3AED")
	ColorSecondary = colorOrDefault(theme.Secondary, "#C4B5FD")
	ColorHighlight = colorOrDefault(theme.Highlight, "#1E1B4B")
	ColorText = colorOrDefault(theme.Text, "#FAFAFA")
	ColorSuccess = colorOrDefault(theme.Success, "#10B981")
	ColorWarning = colorOrDefault(theme.Warning, "#F59E0B")
	ColorMuted = colorOrDefault(theme.Muted, "#6B7280")
	ColorDanger = colorOrDefault(theme.Danger, "#EF4444")
	ColorBorder = colorOrDefault(theme.Border, "#374151")
	ColorDefaultTag = colorOrDefault(theme.DefaultTag, "#312E81")
	buildStyles()
}

func colorOrDefault(value, def string) color.Color {
	if value != "" {
		return lipgloss.Color(colors.Resolve(value))
	}
	return lipgloss.Color(def)
}

// ── Shared styles ─────────────────────────────────────────────────────────────

var (
	// TitleStyle is used for panel title bars.
	TitleStyle lipgloss.Style
	// FilterBarStyle is used for the single-line search bar across all pages.
	FilterBarStyle lipgloss.Style
	// StateOpenStyle highlights the OPEN tunnel state.
	StateOpenStyle lipgloss.Style
	// StateOpeningStyle highlights the transitional OPENING tunnel state.
	StateOpeningStyle lipgloss.Style
	// StateInUseStyle highlights a port held by another process.
	StateInUseStyle lipgloss.Style
	// StateClosedStyle renders the CLOSED / unknown tunnel state.
	StateClosedStyle lipgloss.Style
	// MutedStyle renders secondary information (hints, owners, counters).
	MutedStyle lipgloss.Style
	// ErrorStyle renders inline error messages.
	ErrorStyle lipgloss.Style
	// MatchStyle highlights filter matches in free text.
	MatchStyle lipgloss.Style
	// SectionStyle renders a section heading with a bottom rule.
	SectionStyle lipgloss.Style
	// KeyStyle renders configuration keys / labels.
	KeyStyle lipgloss.Style
	// PanelStyle draws a rounded border around panels.
	PanelStyle lipgloss.Style
	// TagChipStyle renders host tags as small pill-shaped badges.
	TagChipStyle lipgloss.Style
)

func buildStyles() {
	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText).
		Background(ColorPrimary).
		Padding(0, 1)

	FilterBarStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	StateOpenStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)
	StateOpeningStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)
	StateInUseStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorDanger)
	StateClosedStyle = lipgloss.NewStyle().Foreground(ColorMuted)

	MutedStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	ErrorStyle = lipgloss.NewStyle().Foreground(ColorDanger)
	MatchStyle = lipgloss.NewStyle().Foreground(ColorText).Bold(true).Underline(true)

	SectionStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSecondary).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(ColorBorder).
		MarginTop(1)

	KeyStyle = lipgloss.NewStyle().Foreground(ColorSecondary)

	PanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	TagChipStyle = ChipStyle(ColorDefaultTag)
}

// ChipStyle returns a pill-shaped badge style over bg with a readable
// foreground chosen from the background luminance.
func ChipStyle(bg color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(colors.TintedForeground(colors.Hex(bg))).
		Padding(0, 1).
		Bold(true)
}
