package modal

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

// Password is a modal that asks for a secret. Enter submits, esc cancels.
type Password struct {
	title    string
	message  string
	input    textinput.Model
	onSubmit func(password string) tea.Cmd
}

// OpenPassword returns a Cmd that opens a Password modal. onSubmit runs
// after the modal closes with the entered password; cancelling runs nothing.
func OpenPassword(title, message string, onSubmit func(string) tea.Cmd) tea.Cmd {
	in := textinput.New()
	in.EchoMode = textinput.EchoPassword
	in.Prompt = "› "
	in.SetWidth(contentWidth - lipgloss.Width(in.Prompt))
	// The blinking virtual cursor makes the masked value jitter; drop it.
	in.SetVirtualCursor(false)
	return base.Emit(base.OpenModalMsg{
		Dialog: &Password{title: title, message: message, input: in, onSubmit: onSubmit},
	})
}

// Init focuses the input once the modal is registered.
func (m *Password) Init() tea.Cmd {
	return m.input.Focus()
}

// SetSize implements base.Component; the dialog has a fixed width.
func (m *Password) SetSize(_, _ int) {}

// Update implements base.Component.
func (m *Password) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "esc":
			return m, closeModal
		case "enter":
			pw := m.input.Value()
			if strings.TrimSpace(pw) == "" {
				return m, nil
			}
			return m, tea.Sequence(closeModal, m.onSubmit(pw))
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View implements base.Component.
func (m *Password) View() string {
	st := m.input.Styles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(styles.ColorPrimary).Bold(true)
	m.input.SetStyles(st)
	rule := lipgloss.NewStyle().Foreground(styles.ColorBorder).Render(strings.Repeat("─", contentWidth))
	field := m.input.View() + "\n" + rule
	return renderDialog(styles.ColorPrimary,
		"🔒  "+m.title,
		m.message+"\n\n"+field,
		"",
		"enter submit  •  esc cancel",
	)
}
