// Package searchbar provides a single-line search input shared by every page:
// a magnifier glyph, the prompt and a bubbles textinput.
package searchbar

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/glyphs"
)

// icon has a trailing space so the ANSI reset does not land in the middle
// of the 2-cell glyph.
const icon = glyphs.Search + " "

// iconVisualWidth is the width of icon in terminal cells: the Nerd Font PUA
// glyph takes 2 cells plus the trailing space. lipgloss.Width() counts the
// glyph as 1, so this constant is used for layout instead.
const iconVisualWidth = 3

// Model wraps the textinput and renders the bar.
type Model struct {
	input textinput.Model
}

// New creates a search bar with the given prompt, already focused.
func New(prompt string) *Model {
	fi := textinput.New()
	fi.Prompt = prompt
	fi.Focus()
	return &Model{input: fi}
}

// Update processes a bubbletea message, mutates the model in place and returns
// the resulting Cmd along with a boolean indicating whether the text changed.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	prev := m.input.Value()
	newInput, cmd := m.input.Update(msg)
	m.input = newInput
	return cmd, m.input.Value() != prev
}

// View renders the bar on a single line: icon, prompt and input.
func (m *Model) View() string {
	st := m.input.Styles()
	st.Focused.Prompt = styles.MutedStyle
	m.input.SetStyles(st)
	iconStr := styles.MutedStyle.Render(icon)
	return styles.FilterBarStyle.Render(iconStr + m.input.View())
}

// SetWidth adjusts the inner textinput width so the bar fits in totalWidth.
func (m *Model) SetWidth(totalWidth int) {
	promptW := lipgloss.Width(m.input.Prompt)
	inputW := max(0, totalWidth-styles.FilterBarStyle.GetHorizontalFrameSize()-promptW-iconVisualWidth)
	m.input.SetWidth(inputW)
}

// Height returns the rendered height of the bar (for layout calculations).
func (m *Model) Height() int {
	return lipgloss.Height(styles.FilterBarStyle.Render(""))
}

// Value returns the current text of the search field.
func (m *Model) Value() string {
	return m.input.Value()
}

// SetValue sets the text of the search field.
func (m *Model) SetValue(v string) {
	m.input.SetValue(v)
}
