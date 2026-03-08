package modal

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

// Password is a modal dialog with a masked text-input field.
type Password struct {
	title    string
	host     string
	value    []rune
	onResult func(password string, cancelled bool) tea.Cmd
}

func OpenPassword(host string, onResult func(password string, cancelled bool) tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		return base.OpenModalMsg{
			Dialog: &Password{
				title:    "Authentication",
				host:     host,
				onResult: onResult,
			},
		}
	}
}

func (m *Password) SetSize(width int, height int) {
	// TODO
	slog.Debug("Password modal size set", "width", width, "height", height)
}

// Update handles key events: printable chars are appended, Backspace removes
// the last char, Enter confirms, Esc cancels.
func (m *Password) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "enter":
		return m, m.submit(string(m.value), false)
	case "esc":
		return m, m.submit(string(m.value), true)
	case "backspace", "ctrl+h":
		if len(m.value) > 0 {
			m.value = m.value[:len(m.value)-1]
		}
	default:
		// Accept single printable characters only.
		if s := key.String(); len([]rune(s)) == 1 {
			r := []rune(s)[0]
			if r >= 32 && r != 127 {
				m.value = append(m.value, r)
			}
		}
	}
	return m, nil
}

// submit closes the modal and runs onResult.
func (m *Password) submit(password string, cancelled bool) tea.Cmd {
	closeCmd := func() tea.Msg { return base.CloseModalMsg{} }
	return tea.Sequence(closeCmd, m.onResult(password, cancelled))
}

func (m *Password) View() string {
	const dialogWidth = 52

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(styles.ColorPrimary).
		Padding(0, 1).
		Width(dialogWidth - 4)
	title := titleStyle.Render("🔑  " + m.title)

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Width(dialogWidth - 4)
	prompt := promptStyle.Render("Password for " + m.host + ":")

	masked := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#1F2937")).
		Padding(0, 1).
		Width(dialogWidth - 6)

	// Build masked value: show bullets for each typed char + blinking cursor block.
	bullets := ""
	for range m.value {
		bullets += "●"
	}
	inputLine := masked.Render(bullets + "▋")

	hint := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Width(dialogWidth - 4).
		Align(lipgloss.Center).
		Render("enter confirm  •  esc cancel")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary).
		Padding(1, 2).
		Width(dialogWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			prompt,
			"",
			inputLine,
			"",
			hint,
		))
}
