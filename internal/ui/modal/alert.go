package modal

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

// Alert is a modal dialog that shows a message and a single OK button.
type Alert struct {
	title   string
	message string
	isError bool
	onClose func() tea.Cmd // optional callback executed after the modal closes
}

// OpenAlert returns a Cmd that opens an Alert modal.
func OpenAlert(title string, message string, isError bool) tea.Cmd {
	return func() tea.Msg {
		return base.OpenModalMsg{
			Dialog: &Alert{title: title, message: message, isError: isError},
		}
	}
}

func (m *Alert) SetSize(width int, height int) {
	slog.Debug("Alert.SetSize called", "width", width, "height", height)
}

func (m *Alert) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "enter", "esc", " ":
		return m, m.dismiss()
	}
	return m, nil
}

// dismiss closes the modal and runs the optional onClose callback.
func (m *Alert) dismiss() tea.Cmd {
	closeCmd := func() tea.Msg { return base.CloseModalMsg{} }
	if m.onClose != nil {
		return tea.Sequence(closeCmd, m.onClose())
	}
	return closeCmd
}

func (m *Alert) View() string {
	const dialogWidth = 60

	accentColor := styles.ColorPrimary
	icon := "✅  "
	if m.isError {
		accentColor = styles.ColorDanger
		icon = "❌  "
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(accentColor).
		Padding(0, 1).
		Width(dialogWidth - 4)
	title := titleStyle.Render(icon + m.title)

	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5E7EB")).
		Width(dialogWidth - 4)
	body := msgStyle.Render(m.message)

	okStyle := lipgloss.NewStyle().
		Padding(0, 4).
		Bold(true).
		Background(accentColor).
		Foreground(lipgloss.Color("#FFFFFF"))

	okBtn := lipgloss.NewStyle().
		Width(dialogWidth - 4).
		Align(lipgloss.Center).
		Render(okStyle.Render("OK"))

	hint := lipgloss.NewStyle().
		Foreground(styles.ColorMuted).
		Width(dialogWidth - 4).
		Align(lipgloss.Center).
		Render("enter / esc  •  close")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			body,
			"",
			okBtn,
			"",
			hint,
		))
}
