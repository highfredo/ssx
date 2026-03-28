package hosts

import (
	"errors"
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/key"
	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/highfredo/ssx/internal/ftp"
	"github.com/highfredo/ssx/internal/paths"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/system"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/list"
	"github.com/highfredo/ssx/internal/ui/modal"
)

type HostPage struct {
	list      *list.Model
	hosts     []*ssh.HostConfig
	quickHost *ssh.HostConfig
}

var keys = base.Keys().Hosts

func New(hosts []*ssh.HostConfig) *HostPage {
	items := make([]blist.Item, len(hosts))
	for i, h := range hosts {
		items[i] = item{host: h}
	}

	d := hostDelegate{list.NewDefaultDelegate()}

	l := list.New(items, d, 0, 0)
	l.Title = "SSX"
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{keys.Tunnels, keys.FTP}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.Connect, keys.CopyKey, keys.Tunnels, keys.OpenTunnels,
			keys.FTP, keys.Info, keys.OpenSSHDir, keys.OpenConfig,
			keys.QuickConnect,
		}
	}

	m := &HostPage{list: l, hosts: hosts}

	// Quick connect host always should be at the end
	l.Filter = func(term string, targets []string) []blist.Rank {
		if len(targets) == 0 || targets[len(targets)-1] != quickConnectFilterValue {
			return blist.DefaultFilter(term, targets)
		}

		realTargets := targets[:len(targets)-1]
		ranks := blist.DefaultFilter(term, realTargets)
		ranks = append(ranks, blist.Rank{
			Index:          len(targets) - 1,
			MatchedIndexes: []int{},
		})

		return ranks
	}

	return m
}

func (m *HostPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		// Action keys — resolved against the currently selected item.
		item, _ := m.list.SelectedItem().(item)
		switch {
		case key.Matches(keyMsg, keys.FTP):
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
		case key.Matches(keyMsg, keys.QuickConnect):
			if m.quickHost == nil {
				return m, nil
			}
			slog.Info("quick connecting to host", "hostname", m.quickHost.Hostname)
			return m, func() tea.Msg { return OpenShellMsg{config: m.quickHost} }
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
		case key.Matches(keyMsg, keys.OpenSSHDir):
			slog.Info("opening ~/.ssh directory")
			return m, func() tea.Msg {
				if err := system.Open("~/.ssh"); err != nil {
					return modal.OpenAlert("Cannot open ~/.ssh", err.Error(), true)()
				}
				return nil
			}
		case key.Matches(keyMsg, keys.OpenConfig):
			slog.Info("opening ssx config file")
			cfgFile := paths.AppConfigFile()
			return m, func() tea.Msg {
				if err := system.Open(cfgFile); err != nil {
					return modal.OpenAlert("Cannot open config", err.Error(), true)()
				}
				return nil
			}
		}
	}

	switch msg := msg.(type) {
	case OpenShellMsg:
		slog.Info("opening shell", "name", msg.config.Name)
		sshCmd := ssh.Shell(msg.config)
		return m, tea.ExecProcess(sshCmd.Cmd, func(err error) tea.Msg {
			sshCmd.CleanFn()
			if err != nil {
				slog.Error("error opening shell", "error", err)
				return tea.Quit()
			}

			return nil
		})

	case SshCopyIdMsg:
		sshCmd, err := ssh.CopyId(msg.config)
		if err != nil {
			slog.Error("ssh copy-id: no public key", "error", err)
			return m, modal.OpenAlert("No public key found", err.Error(), true)
		}
		return m, tea.ExecProcess(sshCmd.Cmd, func(err error) tea.Msg {
			sshCmd.CleanFn()
			if err != nil {
				slog.Error("ssh copy-id failed", "host", msg.config.Hostname, "error", err)
				return tea.Quit()
			}
			slog.Info("ssh copy-id succeeded", "host", msg.config.Hostname)
			return modal.OpenAlert(
				"Key copied successfully",
				fmt.Sprintf("The public key was copied to %s.", msg.config.Hostname),
				false,
			)
		})
	}

	var cmds []tea.Cmd
	oldSearchVal := m.list.SearchValue()
	listCmd := m.list.Update(msg)

	if oldSearchVal != m.list.SearchValue() {
		itemsLen := len(m.list.Items())
		if con := ssh.ParseConnection(m.list.SearchValue()); con != nil {
			con.Title = "⚡Quick Connect"
			m.quickHost = con
			quickHost := item{host: con, isQuickConnect: true}
			var insCmd tea.Cmd
			if len(m.hosts) == itemsLen {
				insCmd = m.list.InsertItem(itemsLen, quickHost)
			} else {
				insCmd = m.list.SetItem(itemsLen-1, quickHost)
			}

			cmds = append(cmds, insCmd)
		} else if m.list.Items()[itemsLen-1].(item).isQuickConnect {
			m.list.RemoveItem(itemsLen - 1)
			m.quickHost = nil
		}
	}

	cmds = append(cmds, listCmd)
	return m, tea.Sequence(cmds...)
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
