package modal

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

// Spinner is a non-interactive modal that animates while a slow operation
// runs in the background. It is closed externally with base.CloseModalMsg.
type Spinner struct {
	message string
	spin    spinner.Model
}

// OpenSpinner returns a Cmd that opens the spinner modal. The App starts the
// tick chain by calling Init once the modal is registered.
func OpenSpinner(message string) tea.Cmd {
	s := &Spinner{
		message: message,
		spin:    spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorPrimary))),
	}
	return base.Emit(base.OpenModalMsg{Dialog: s})
}

// Init starts the spinner tick chain. The App calls it right after
// registering the modal so no TickMsg arrives before the modal is open.
func (m *Spinner) Init() tea.Cmd {
	return m.spin.Tick
}

// SetSize implements base.Component; the spinner has a fixed size.
func (m *Spinner) SetSize(_, _ int) {}

// Update implements base.Component.
func (m *Spinner) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

// View implements base.Component.
func (m *Spinner) View() string {
	return styles.PanelStyle.
		BorderForeground(styles.ColorPrimary).
		Padding(1, 3).
		Render(m.spin.View() + "  " + m.message)
}
