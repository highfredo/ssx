package modal

import (
	"log/slog"
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
	choice   int // 0 = Yes, 1 = No
	onResult func(confirmed bool) tea.Cmd
}

func OpenConfirm(title string, message string, onResult func(bool) tea.Cmd) tea.Cmd {
	return func() tea.Msg {
		return base.OpenModalMsg{
			Dialog: &Confirm{title: title, message: message, choice: 0, onResult: onResult},
		}
	}
}

func (m *Confirm) SetSize(width int, height int) {
	//TODO implement me
	slog.Debug("Confirm.SetSize called", "width", width, "height", height)
}

func (m *Confirm) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "left", "h", "shift+tab":
		m.choice = 0
	case "right", "l", "tab":
		m.choice = 1
	case "y", "Y":
		return m, m.confirm(true)
	case "n", "N", "esc":
		return m, m.confirm(false)
	case "enter":
		return m, m.confirm(m.choice == 0)
	}
	return m, nil
}

// confirm closes the modal and runs onResult.
func (m *Confirm) confirm(confirmed bool) tea.Cmd {
	closeCmd := func() tea.Msg { return base.CloseModalMsg{} }
	return tea.Sequence(closeCmd, m.onResult(confirmed))
}

func (m *Confirm) View() string {
	const dialogWidth = 60

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(styles.ColorDanger).
		Padding(0, 1).
		Width(dialogWidth - 4)
	title := titleStyle.Render("\u26a0\ufe0f  " + m.title)

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Width(dialogWidth - 4)
	body := msgStyle.Render(m.message)

	yesStyle := lipgloss.NewStyle().Padding(0, 2).Bold(true)
	noStyle := lipgloss.NewStyle().Padding(0, 2).Bold(true)

	if m.choice == 0 {
		yesStyle = yesStyle.
			Background(styles.ColorDanger).
			Foreground(lipgloss.Color("#FFFFFF"))
		noStyle = noStyle.
			Background(styles.ColorBorder).
			Foreground(lipgloss.Color("#9CA3AF"))
	} else {
		yesStyle = yesStyle.
			Background(styles.ColorBorder).
			Foreground(lipgloss.Color("#9CA3AF"))
		noStyle = noStyle.
			Background(styles.ColorDanger).
			Foreground(lipgloss.Color("#FFFFFF"))
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("Yes"),
		strings.Repeat(" ", 4),
		noStyle.Render("No"),
	)
	buttonsRow := lipgloss.NewStyle().Width(dialogWidth - 4).Align(lipgloss.Center).Render(buttons)

	hint := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Width(dialogWidth - 4).
		Align(lipgloss.Center).
		Render("\u2190/\u2192 navigate  \u2022  enter confirm  \u2022  esc cancel")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorDanger).
		Padding(1, 2).
		Width(dialogWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			body,
			"",
			buttonsRow,
			"",
			hint,
		))
}
