package ui

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/hosts"
	"github.com/highfredo/ssx/internal/ui/info"
	"github.com/highfredo/ssx/internal/ui/tunnels"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type App struct {
	rootPage base.Component

	// activeScreen es lo que el usuario ve e interactúa actualmente.
	activePage base.Component

	// Services
	TunnelManager *ssh.TunnelManager

	width  int
	height int

	modal base.Component
}

func NewApp(tunnelManager *ssh.TunnelManager) *App {
	hostconfigs, err := ssh.LoadConfig()
	if err != nil {
		panic(err)
	}
	// Al inicio, la pantalla activa es la principal
	hostPage := hosts.New(hostconfigs)
	return &App{
		activePage:    hostPage,
		rootPage:      hostPage,
		TunnelManager: tunnelManager,
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return nil
}

// Update maneja la lógica de orquestación y delega eventos a la pantalla activa.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		a.width = msg.Width - h
		a.height = msg.Height - v
		a.activePage.SetSize(a.width, a.height)
		if a.modal != nil {
			a.modal.SetSize(a.width, a.height)
		}
		return a, nil
	case base.OpenTunnelForHostPageMsg:
		title := "Tunnels for " + msg.HostConfig.Name
		tl := tunnels.New(title, msg.HostConfig.Tunnels, a.TunnelManager)
		tl.SetSize(a.width, a.height)
		a.activePage = tl
		return a, nil
	case base.OpenTunnelsOpenPageMsg:
		tunnelsOpen := a.TunnelManager.OpenTunnels(ssh.GetHosts())
		tl := tunnels.New("Opens Tunnels", tunnelsOpen, a.TunnelManager)
		tl.SetSize(a.width, a.height)
		a.activePage = tl
		return a, nil
	case base.OpenInfoPageMsg:
		ip := info.New(msg.HostConfig)
		ip.SetSize(a.width, a.height)
		a.activePage = ip
		return a, nil
	case base.OpenHostPageMsg:
		a.activePage = a.rootPage
		a.activePage.SetSize(a.width, a.height)
		return a, nil
	case base.OpenModalMsg:
		slog.Debug("opening modal")
		a.modal = msg.Dialog
		// Si el modal tiene Init(), ejecutarlo ahora que ya está registrado.
		// Esto garantiza que el primer tick del spinner llega después de que
		// a.modal != nil, evitando la carrera que mata la cadena de ticks.
		type initable interface{ Init() tea.Cmd }
		if init, ok := msg.Dialog.(initable); ok {
			return a, init.Init()
		}
		return a, nil
	case base.CloseModalMsg:
		slog.Debug("closing modal")
		a.modal = nil
		return a, nil
	}

	if a.modal != nil {
		model, cmd := a.modal.Update(msg)
		a.modal = model
		return a, cmd
	}

	model, cmd := a.activePage.Update(msg)
	a.activePage = model

	return a, cmd
}

// View delega el renderizado a la pantalla activa y superpone modales.
func (a App) View() tea.View {
	content := a.activePage.View()

	if a.modal != nil {
		dialog := a.modal.View()
		dw := lipgloss.Width(dialog)
		dh := lipgloss.Height(dialog)

		x := (a.width - dw) / 2
		y := (a.height - dh) / 2
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}

		bgLayer := lipgloss.NewLayer(content)
		fgLayer := lipgloss.NewLayer(dialog).X(x).Y(y).Z(1)

		content = lipgloss.NewCompositor(bgLayer, fgLayer).Render()
	}

	v := tea.NewView(docStyle.Render(content))
	v.AltScreen = true
	return v
}
