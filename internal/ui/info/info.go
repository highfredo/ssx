// Package info shows the effective ssh configuration of a host (ssh -G),
// grouped by topic and searchable.
package info

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/colors"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/list"
	"github.com/highfredo/ssx/internal/ui/searchbar"
)

// InfoPage shows the effective SSH configuration for a host (ssh -G).
type InfoPage struct {
	host     *ssh.HostConfig
	allLines []parsedLine
	viewport viewport.Model
	search   *searchbar.Model
	help     help.Model
	keys     base.NavigationKeys
	width    int
	height   int
}

// ShortHelp implements help.KeyMap.
func (p *InfoPage) ShortHelp() []key.Binding {
	return []key.Binding{p.keys.Back, p.keys.ShowFullHelp}
}

// FullHelp implements help.KeyMap.
func (p *InfoPage) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{p.keys.Back},
		{p.keys.HalfPageUp, p.keys.HalfPageDown},
		{p.keys.ShowFullHelp},
	}
}

// New runs ssh -G for host and builds the page.
func New(host *ssh.HostConfig, keys base.NavigationKeys) *InfoPage {
	lines := append(buildMagicLines(host), parseSSHG(host)...)

	vp := viewport.New()
	vp.SoftWrap = true
	vp.KeyMap.HalfPageUp = keys.HalfPageUp
	vp.KeyMap.HalfPageDown = keys.HalfPageDown
	vp.KeyMap.PageUp = keys.PrevPage
	vp.KeyMap.PageDown = keys.NextPage
	vp.KeyMap.Up = keys.CursorUp
	vp.KeyMap.Down = keys.CursorDown

	p := &InfoPage{
		host:     host,
		allLines: lines,
		viewport: vp,
		search:   searchbar.New("Search: "),
		help:     help.New(),
		keys:     keys,
	}
	p.refreshViewport()
	return p
}

// Update implements base.Component.
func (p *InfoPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, p.keys.Back):
			// esc clears the filter first; a second press navigates back.
			if p.search.Value() != "" {
				p.search.SetValue("")
				p.refreshViewport()
				return p, nil
			}
			return p, base.Emit(base.OpenHostPageMsg{})
		case key.Matches(keyMsg, p.keys.ShowFullHelp):
			p.help.ShowAll = !p.help.ShowAll
			return p, nil
		}
	}

	var vpCmd tea.Cmd
	p.viewport, vpCmd = p.viewport.Update(msg)

	searchCmd, changed := p.search.Update(msg)
	if changed {
		p.refreshViewport()
	}
	return p, tea.Batch(vpCmd, searchCmd)
}

// SetSize implements base.Component.
func (p *InfoPage) SetSize(w, h int) {
	p.width, p.height = w, h
	if w < list.NarrowWidth {
		p.help.ShowAll = false
	}
	p.search.SetWidth(w)
	p.viewport.SetWidth(w)
	p.help.SetWidth(w)
}

// View implements base.Component.
func (p *InfoPage) View() string {
	title := styles.TitleStyle.Render("Config · " + p.host.Name)
	search := p.search.View()
	helpView := p.help.View(p)
	if !p.help.ShowAll {
		helpView = list.TruncateLine(helpView, p.width)
	}

	overhead := lipgloss.Height(title) + lipgloss.Height(search) + lipgloss.Height(helpView)
	vpH := max(p.height-overhead, 1)
	p.viewport.SetHeight(vpH)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		search,
		p.viewport.View(),
		helpView,
	)
}

// refreshViewport rebuilds the viewport content applying the current filter.
func (p *InfoPage) refreshViewport() {
	q := strings.ToLower(p.search.Value())

	var sb strings.Builder
	for _, sec := range groupLines(p.allLines, q) {
		sb.WriteString(styles.SectionStyle.Render(sec.title) + "\n")
		for _, l := range sec.lines {
			sb.WriteString(highlightMatchStyled(l.key, q, styles.KeyStyle) + " " + p.renderValue(l, q) + "\n")
		}
	}

	p.viewport.SetContent(sb.String())
	p.viewport.GotoTop()
}

// renderValue draws a line value, using tag chips when the line carries tags.
func (p *InfoPage) renderValue(l parsedLine, q string) string {
	if len(l.tags) == 0 {
		return highlightMatch(l.value, q)
	}
	chips := make([]string, len(l.tags))
	for i, t := range l.tags {
		chip := styles.ChipStyle(lipgloss.Color(colors.Resolve(t.Color)))
		// Highlight inline so the chip padding/background is rendered once.
		chips[i] = chip.Render(highlightMatchStyled(t.Name, q, chip.Inline(true)))
	}
	return strings.Join(chips, " ")
}

// highlightMatch highlights all occurrences of q in s (case-insensitive).
func highlightMatch(s, q string) string {
	return highlightMatchStyled(s, q, lipgloss.NewStyle())
}

// highlightMatchStyled searches for q in the raw text s and applies:
//   - base style to non-matching parts
//   - matchStyle to matching parts
//
// Searching on the raw string (no ANSI) ensures correct byte indices.
func highlightMatchStyled(s, q string, base lipgloss.Style) string {
	if q == "" {
		return base.Render(s)
	}
	lower := strings.ToLower(s)
	var out strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], q)
		if idx < 0 {
			out.WriteString(base.Render(s[i:]))
			break
		}
		if idx > 0 {
			out.WriteString(base.Render(s[i : i+idx]))
		}
		out.WriteString(base.Inherit(styles.MatchStyle).Render(s[i+idx : i+idx+len(q)]))
		i += idx + len(q)
	}
	return out.String()
}
