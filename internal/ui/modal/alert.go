package modal

import (
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

// Alert is a modal dialog that shows a message and a single OK button.
type Alert struct {
	title   string
	message string
	isError bool
}

// OpenAlert returns a Cmd that opens an Alert modal.
func OpenAlert(title, message string, isError bool) tea.Cmd {
	return base.Emit(base.OpenModalMsg{
		Dialog: &Alert{title: title, message: message, isError: isError},
	})
}

// SetSize implements base.Component; the dialog has a fixed width.
func (m *Alert) SetSize(_, _ int) {}

// Update implements base.Component.
func (m *Alert) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "enter", "esc", " ":
		return m, closeModal
	}
	return m, nil
}

// View implements base.Component.
func (m *Alert) View() string {
	accent, icon := styles.ColorSuccess, "✔  "
	if m.isError {
		accent, icon = styles.ColorDanger, "✖  "
	}
	return renderDialog(accent,
		icon+m.title,
		m.message,
		renderButton("  OK  ", true, accent),
		"enter / esc  •  close",
	)
}
