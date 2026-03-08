package hosts

import (
	"errors"
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/key"
	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/ftp"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/list"
	"github.com/highfredo/ssx/internal/ui/modal"
)

var keys = struct {
	Connect     key.Binding
	CopyKey     key.Binding
	Tunnels     key.Binding
	OpenTunnels key.Binding
	WinSCP      key.Binding
	Info        key.Binding
}{
	Connect:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
	CopyKey:     key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "copy SSH key")),
	Tunnels:     key.NewBinding(key.WithKeys("ctrl+t", "tab"), key.WithHelp("ctrl+t/tab", "tunnels")),
	OpenTunnels: key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "open tunnels")),
	WinSCP:      key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "WinSCP")),
	Info:        key.NewBinding(key.WithKeys("ctrl+g"), key.WithHelp("ctrl+g", "ssh info")),
}

type HostPage struct {
	list  *list.Model
	hosts []*ssh.HostConfig
}

func New(hosts []*ssh.HostConfig) *HostPage {
	items := make([]blist.Item, len(hosts))
	for i, h := range hosts {
		items[i] = item{host: h}
	}

	d := hostDelegate{list.NewDefaultDelegate()}

	l := list.New(items, d, 0, 0)
	l.Title = "SSX"
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Connect, keys.CopyKey, keys.Tunnels, keys.Info}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Connect, keys.CopyKey, keys.Tunnels, keys.OpenTunnels, keys.WinSCP, keys.Info}
	}

	return &HostPage{list: l, hosts: hosts}
}

func (m *HostPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		// Action keys — resolved against the currently selected item.
		item, _ := m.list.SelectedItem().(item)
		switch {
		case key.Matches(keyMsg, keys.WinSCP):
			slog.Info("opening SFTP client", "name", item.host.Name)
			h := item.host
			return m, tea.Batch(
				modal.OpenSpinner("Opening SFTP client…"),
				func() tea.Msg {
					if err := ftp.Launch(h); errors.Is(err, ftp.ErrNoFTPClient) {
						// OpenModalMsg reemplaza el spinner con la alerta;
						// el App lo maneja antes del routing al modal.
						return modal.OpenAlert("No SFTP client found", "Please install WinSCP or FileZilla", true)()
					}
					// CloseModalMsg también lo maneja el App directamente.
					return base.CloseModalMsg{}
				},
			)
		case key.Matches(keyMsg, keys.OpenTunnels):
			slog.Info("opening tunnels open page")
			return m, func() tea.Msg { return base.OpenTunnelsOpenPageMsg{} }
		case key.Matches(keyMsg, keys.Tunnels):
			slog.Info("opening tunnels page", "name", item.host.Name)
			return m, func() tea.Msg { return base.OpenTunnelForHostPageMsg{HostConfig: item.host} }
		case key.Matches(keyMsg, keys.Connect):
			slog.Info("connecting to host", "name", item.host.Name)
			return m, func() tea.Msg {
				return OpenShellMsg{config: item.host}
			}
		case key.Matches(keyMsg, keys.CopyKey):
			slog.Info("ssh-copy-id confirm", "name", item.host.Name)
			h := item.host
			return m, modal.OpenConfirm(
				"Copy SSH key",
				fmt.Sprintf("Copy your public key to %s (%s)?", h.Name, h.Hostname),
				func(confirmed bool) tea.Cmd {
					if !confirmed {
						return nil
					}
					return func() tea.Msg { return SshCopyIdMsg{config: h} }
				},
			)
		case key.Matches(keyMsg, keys.Info):
			slog.Info("opening ssh info", "name", item.host.Name)
			return m, func() tea.Msg { return base.OpenInfoPageMsg{HostConfig: item.host} }
		}
	}

	switch msg := msg.(type) {
	case OpenShellMsg:
		slog.Error("abriendo shell", "name", msg.config.Name)
		cmd := ssh.Shell(msg.config)
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			// FIXME no todo error es que la contrseña este mal
			if err != nil {
				slog.Error("error abriendo shell", "error", err)
				return modal.OpenPassword(msg.config.Name, func(password string, cancelled bool) tea.Cmd {
					if cancelled {
						return nil
					}
					msg.config.Password = password
					return func() tea.Msg { return OpenShellMsg{config: msg.config} }
				})
			}

			return tea.Quit()
		})

	case SshCopyIdMsg:
		askPassword := func() tea.Msg {
			return modal.OpenPassword(msg.config.Hostname, func(password string, cancelled bool) tea.Cmd {
				if cancelled {
					return nil
				}
				msg.config.Password = password
				return func() tea.Msg { return SshCopyIdMsg{config: msg.config} }
			})
		}

		if msg.config.Password == "" && msg.config.PasswordCommand == "" {
			return m, func() tea.Msg { return askPassword() }
		}

		cmd, err := ssh.CopyId(msg.config)
		if err != nil {
			slog.Error("ssh copy-id: no public key", "error", err)
			return m, modal.OpenAlert("No public key found", err.Error(), true)
		}
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			if err != nil {
				slog.Error("ssh copy-id failed", "host", msg.config.Hostname, "error", err)
				msg.config.Password = ""
				return askPassword()
			}
			slog.Info("ssh copy-id succeeded", "host", msg.config.Hostname)
			return modal.OpenAlert(
				"Key copied successfully",
				fmt.Sprintf("The public key was copied to %s.", msg.config.Hostname),
				false,
			)
		})
	}

	return m, m.list.Update(msg)
}

func (m *HostPage) SetSize(w, h int) {
	slog.Info("resizing host list", "width", w, "height", h)
	m.list.SetSize(w, h)
}

func (m *HostPage) View() string {
	return m.list.View()
}

/***
* Messages
 */

type OpenShellMsg struct {
	config *ssh.HostConfig
}

type SshCopyIdMsg struct {
	config *ssh.HostConfig
}
