package base

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"github.com/highfredo/ssx/internal/appconfig"
)

// resolve creates a key.Binding from a user-configured string of space-separated
// keys, falling back to defaultKeys if empty. The help key is auto-derived.
func resolve(configured, defaultKeys, helpDesc string) key.Binding {
	ks := strings.Fields(configured)
	if len(ks) == 0 {
		ks = strings.Fields(defaultKeys)
	}
	return key.NewBinding(key.WithKeys(ks...), key.WithHelp(strings.Join(ks, "/"), helpDesc))
}

// AppKeybindings holds all resolved key bindings for the application.
// Initialized once from appconfig — use Keys() to access.
type AppKeybindings struct {
	Quit       key.Binding
	Hosts      HostsKeys
	Tunnels    TunnelsKeys
	Navigation NavigationKeys
}

// HostsKeys holds action bindings for the host list page.
type HostsKeys struct {
	Connect      key.Binding
	CopyKey      key.Binding
	Tunnels      key.Binding
	OpenTunnels  key.Binding
	FTP          key.Binding
	Info         key.Binding
	OpenSSHDir   key.Binding
	OpenConfig   key.Binding
	QuickConnect key.Binding
}

// TunnelsKeys holds action bindings for the tunnel page.
type TunnelsKeys struct {
	Toggle  key.Binding
	Browser key.Binding
	//Back    key.Binding
}

// NavigationKeys holds navigation bindings shared by all lists and scroll views.
type NavigationKeys struct {
	CursorUp      key.Binding
	CursorDown    key.Binding
	PrevPage      key.Binding
	NextPage      key.Binding
	GoToStart     key.Binding
	GoToEnd       key.Binding
	ShowFullHelp  key.Binding
	CloseFullHelp key.Binding
	HalfPageUp    key.Binding
	HalfPageDown  key.Binding
	Back          key.Binding
}

// cachedKeys is the application-wide keybindings singleton.
var cachedKeys = buildKeys(appconfig.Get().Keybindings)

// Keys returns the application-wide resolved key bindings singleton.
func Keys() *AppKeybindings {
	return cachedKeys
}

func buildKeys(cfg appconfig.KeybindingsConfig) *AppKeybindings {
	h := cfg.Hosts
	t := cfg.Tunnels
	n := cfg.Navigation

	return &AppKeybindings{
		Quit: resolve(cfg.Quit, "ctrl+c", "quit"),
		Hosts: HostsKeys{
			Connect:      resolve(h.Connect, "enter", "connect"),
			CopyKey:      resolve(h.CopyKey, "ctrl+a", "copy SSH key"),
			Tunnels:      resolve(h.Tunnels, "ctrl+t tab", "tunnels"),
			OpenTunnels:  resolve(h.OpenTunnels, "ctrl+o", "opened tunnels"),
			FTP:          resolve(h.FTP, "ctrl+x", "open FTP"),
			Info:         resolve(h.Info, "ctrl+g", "ssh info"),
			OpenSSHDir:   resolve(h.OpenSSHDir, "ctrl+alt+p", "open ssh config"),
			OpenConfig:   resolve(h.OpenConfig, "ctrl+alt+s", "open settings"),
			QuickConnect: resolve(h.QuickConnect, "ctrl+enter ctrl+j", "quick connect"),
		},
		Tunnels: TunnelsKeys{
			Toggle:  resolve(t.Toggle, "enter", "toggle tunnel"),
			Browser: resolve(t.Browser, "ctrl+x", "open in browser"),
			//Back:    resolve(t.Back, "esc tab", "back"),
		},
		Navigation: NavigationKeys{
			CursorUp:      resolve(n.CursorUp, "up", "up"),
			CursorDown:    resolve(n.CursorDown, "down", "down"),
			PrevPage:      resolve(n.PrevPage, "pgup", "prev page"),
			NextPage:      resolve(n.NextPage, "pgdown", "next page"),
			GoToStart:     resolve(n.GoToStart, "home", "go to start"),
			GoToEnd:       resolve(n.GoToEnd, "end", "go to end"),
			ShowFullHelp:  resolve(n.ShowFullHelp, "f1", "open help"),
			CloseFullHelp: resolve(n.ShowFullHelp, "f1", "close help"),
			HalfPageUp:    resolve(n.HalfPageUp, "ctrl+u", "½ page up"),
			HalfPageDown:  resolve(n.HalfPageDown, "ctrl+d", "½ page down"),
			Back:          resolve(n.Back, "tab esc", "back"),
		},
	}
}
