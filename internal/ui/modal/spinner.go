package modal

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
)

var spinnerStyle = lipgloss.NewStyle().Foreground(styles.ColorPrimary)

// Spinner es un modal no interactivo que muestra una animación mientras
// una operación lenta se ejecuta en segundo plano.
// Se cierra externamente mediante base.CloseModalMsg.
type Spinner struct {
	message string
	spin    spinner.Model
}

// OpenSpinner devuelve un Cmd que abre el modal spinner.
// El primer tick lo inicia el App cuando llama a Init() al registrar el modal.
func OpenSpinner(message string) tea.Cmd {
	s := &Spinner{
		message: message,
		spin:    spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(spinnerStyle)),
	}
	return func() tea.Msg { return base.OpenModalMsg{Dialog: s} }
}

// Init arranca la cadena de ticks del spinner.
// El App lo llama justo después de registrar el modal, evitando la carrera
// donde el TickMsg llega antes de que el modal esté abierto.
func (m *Spinner) Init() tea.Cmd {
	return m.spin.Tick
}

func (m *Spinner) SetSize(_, _ int) {}

func (m *Spinner) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m *Spinner) View() string {
	content := m.spin.View() + "  " + m.message
	return styles.PanelStyle.
		BorderForeground(styles.ColorPrimary).
		Padding(1, 3).
		Render(content)
}
