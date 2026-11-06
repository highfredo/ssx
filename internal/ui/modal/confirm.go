package modal

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

// Confirm is a modal dialog with Yes / No buttons.
type Confirm struct {
	title    string
	message  string
	yes      bool // currently highlighted choice
	onResult func(confirmed bool) tea.Cmd
	danger   bool // destructive action: warning accent
}

// OpenConfirm returns a Cmd that opens a Confirm modal. onResult runs after
// the modal closes with the user's answer.
func OpenConfirm(title, message string, onResult func(bool) tea.Cmd) tea.Cmd {
	return base.Emit(base.OpenModalMsg{
		Dialog: &Confirm{title: title, message: message, yes: true, onResult: onResult},
	})
}

// OpenDangerConfirm is like OpenConfirm but styled for destructive actions,
// defaulting the highlighted choice to "No".
func OpenDangerConfirm(title, message string, onResult func(bool) tea.Cmd) tea.Cmd {
	return base.Emit(base.OpenModalMsg{
		Dialog: &Confirm{title: title, message: message, yes: false, danger: true, onResult: onResult},
	})
}

// SetSize implements base.Component; the dialog has a fixed width.
func (m *Confirm) SetSize(_, _ int) {}

// Update implements base.Component.
func (m *Confirm) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "left", "h", "shift+tab":
		m.yes = true
	case "right", "l", "tab":
		m.yes = false
	case "y", "Y":
		return m, m.answer(true)
	case "n", "N", "esc":
		return m, m.answer(false)
	case "enter":
		return m, m.answer(m.yes)
	}
	return m, nil
}

// answer closes the modal and runs onResult.
func (m *Confirm) answer(confirmed bool) tea.Cmd {
	return tea.Sequence(closeModal, m.onResult(confirmed))
}

// View implements base.Component.
func (m *Confirm) View() string {
	accent, icon := styles.ColorPrimary, "?  "
	if m.danger {
		accent, icon = styles.ColorWarning, "⚠  "
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		renderButton("Yes", m.yes, accent),
		strings.Repeat(" ", 4),
		renderButton("No", !m.yes, accent),
	)
	return renderDialog(accent,
		icon+m.title,
		m.message,
		buttons,
		"←/→ navigate  •  enter confirm  •  esc cancel",
	)
}
