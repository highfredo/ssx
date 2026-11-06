// Package ui implements the Bubble Tea application that drives ssx.
//
// App acts as a router: it owns the currently active page, the host list used
// as root page and an optional modal overlay. Pages communicate through the
// messages declared in package base.
package ui

import (
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/appconfig"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/hosts"
	"github.com/highfredo/ssx/internal/ui/info"
	"github.com/highfredo/ssx/internal/ui/modal"
	"github.com/highfredo/ssx/internal/ui/tunnels"
)

var docStyle = lipgloss.NewStyle().Margin(0, 1)

// App is the root Bubble Tea model.
type App struct {
	hosts ssh.Hosts
	cfg   *appconfig.Config
	keys  *base.AppKeybindings

	rootPage   base.Component // host list, shown on start and when navigating back
	activePage base.Component // page the user currently sees and interacts with
	modal      base.Component // optional overlay that captures all input while open

	width  int
	height int
}

// NewApp creates the application with the host list as the initial page.
// The theme must already be applied (see styles.Apply) since pages read
// styles on construction.
func NewApp(hostList ssh.Hosts, cfg *appconfig.Config, keys *base.AppKeybindings) *App {
	hostPage := hosts.New(hostList, cfg.Tags, keys)
	return &App{
		hosts:      hostList,
		cfg:        cfg,
		keys:       keys,
		rootPage:   hostPage,
		activePage: hostPage,
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return nil
}

// Update handles navigation and cross-page actions, then delegates any other
// message to the modal (if open) or to the active page.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, a.keys.Quit) {
			return a, tea.Quit
		}

	case tea.WindowSizeMsg:
		return a.resize(msg), nil

	// Navigation.
	case base.OpenTunnelForHostPageMsg:
		return a.showPage(tunnels.NewForHost(msg.HostConfig, a.keys))
	case base.OpenTunnelsOpenPageMsg:
		return a.showPage(tunnels.NewOpen(a.hosts, a.keys))
	case base.OpenInfoPageMsg:
		return a.showPage(info.New(msg.HostConfig, a.keys.Navigation))
	case base.OpenHostPageMsg:
		return a.showPage(a.rootPage)

	// Host actions shared by several pages.
	case base.CopySSHKeyMsg:
		return a, confirmCopySSHKey(msg.HostConfig)
	case copySSHKeyMsg:
		return a, copySSHKey(msg.host)
	case base.OpenFTPMsg:
		return a, openFTP(msg.HostConfig, a.cfg.FTP)
	case base.EditSSHConfigMsg:
		return a, editSSHConfig(msg.HostConfig)

	case base.AppUpdatedMsg:
		title := fmt.Sprintf("✦ ssx updated to %s", msg.NewVersion)
		return a, modal.OpenAlert(title, "Enjoy!", false)

	// Modal lifecycle.
	case base.OpenModalMsg:
		slog.Debug("opening modal")
		a.modal = msg.Dialog
		a.modal.SetSize(a.width, a.height)
		return a, initComponent(msg.Dialog)
	case base.CloseModalMsg:
		slog.Debug("closing modal")
		a.modal = nil
		return a, nil
	}

	var cmd tea.Cmd
	if a.modal != nil {
		a.modal, cmd = a.modal.Update(msg)
		return a, cmd
	}
	a.activePage, cmd = a.activePage.Update(msg)
	return a, cmd
}

func (a App) resize(msg tea.WindowSizeMsg) App {
	h, v := docStyle.GetFrameSize()
	a.width = msg.Width - h
	a.height = msg.Height - v
	a.activePage.SetSize(a.width, a.height)
	if a.modal != nil {
		a.modal.SetSize(a.width, a.height)
	}
	return a
}

// showPage makes page the active one, sizing and initialising it.
func (a App) showPage(page base.Component) (tea.Model, tea.Cmd) {
	page.SetSize(a.width, a.height)
	a.activePage = page
	return a, initComponent(page)
}

// initComponent runs the component's Init, if it defines one. Components are
// not required to implement tea.Model in full.
func initComponent(component base.Component) tea.Cmd {
	if init, ok := component.(interface{ Init() tea.Cmd }); ok {
		return init.Init()
	}
	return nil
}

// View renders the active page and, when open, centres the modal on top.
func (a App) View() tea.View {
	content := a.activePage.View()
	if a.modal != nil {
		content = overlayCentered(content, a.modal.View(), a.width, a.height)
	}

	v := tea.NewView(docStyle.Render(content))
	v.AltScreen = true
	return v
}

func overlayCentered(background, dialog string, width, height int) string {
	x := max(0, (width-lipgloss.Width(dialog))/2)
	y := max(0, (height-lipgloss.Height(dialog))/2)

	bg := lipgloss.NewLayer(background)
	fg := lipgloss.NewLayer(dialog).X(x).Y(y).Z(1)
	return lipgloss.NewCompositor(bg, fg).Render()
}
