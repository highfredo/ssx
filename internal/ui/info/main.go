package info

import (
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/ssh"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
	"github.com/highfredo/ssx/internal/ui/searchbar"
)

var keys = base.Keys().Navigation

var (
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	matchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Bold(true).Underline(true)
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#C4B5FD")).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("#4C1D95")).
			MarginTop(1)
)

// sectionDef agrupa claves de ssh -G bajo un título temático.
type sectionDef struct {
	title string
	keys  []string // claves en minúsculas tal como las imprime ssh -G
}

var sshSections = []sectionDef{
	{
		title: "🔌 Connection",
		keys: []string{
			"hostname", "port", "user", "addressfamily",
			"connecttimeout", "connectionattempts",
			"serveralivecountmax", "serveraliveinterval", "tcpkeepalive",
		},
	},
	{
		title: "⟳ Forwarding",
		keys: []string{
			"localforward", "remoteforward", "dynamicforward",
			"forwardagent", "forwardx11", "forwardx11timeout", "forwardx11trusted",
			"gatewayports", "clearallforwardings", "permitremoteopen", "permittunnel",
		},
	},
	{
		title: "🔀 Proxy",
		keys:  []string{"proxycommand", "proxyjump"},
	},
	{
		title: "🔐 Authentication",
		keys: []string{
			"identityfile", "identityagent", "certificate",
			"preferredauthentications", "pubkeyauthentication",
			"passwordauthentication", "kbdinteractiveauthentication",
			"hostbasedauthentication", "gssapiauthentication",
			"numberofpasswordprompts",
		},
	},
	{
		title: "🔒 Cryptography",
		keys: []string{
			"ciphers", "macs", "hostkeyalgorithms", "kexalgorithms",
			"pubkeyacceptedalgorithms", "casignaturealgorithms",
			"fingerprinthash", "hostbasedacceptedalgorithms",
		},
	},
	{
		title: "🛡️ Host Verification",
		keys: []string{
			"stricthostkeychecking", "userknownhostsfile", "globalknownhostsfile",
			"checkhostip", "verifyhostkeydns",
		},
	},
	{
		title: "⚡ Multiplexing",
		keys:  []string{"controlmaster", "controlpath", "controlpersist"},
	},
	{
		title: "🖥️ Session",
		keys: []string{
			"requesttty", "sessiontype", "permittylocalcommand", "localcommand",
			"sendenv", "setenv", "remotecommand",
		},
	},
}

// parsedLine almacena una línea de ssh -G ya descompuesta.
type parsedLine struct {
	key   string
	value string
}

// InfoPage muestra la configuración efectiva de SSH para un host (ssh -G).
type InfoPage struct {
	host     *ssh.HostConfig
	allLines []parsedLine
	viewport viewport.Model
	search   *searchbar.Model
	help     help.Model
	width    int
	height   int
}

func (p *InfoPage) ShortHelp() []key.Binding {
	return []key.Binding{keys.Back, keys.CursorUp, keys.CursorDown, keys.ShowFullHelp}
}

func (p *InfoPage) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keys.Back},
		{keys.CursorUp, keys.CursorDown, keys.PrevPage, keys.NextPage, keys.HalfPageUp, keys.HalfPageDown},
		{keys.ShowFullHelp},
	}
}

func New(host *ssh.HostConfig) *InfoPage {
	lines := parseSSHG(host)

	vp := viewport.New()
	vp.SoftWrap = true
	vp.KeyMap.HalfPageUp = keys.HalfPageUp
	vp.KeyMap.HalfPageDown = keys.HalfPageDown
	vp.KeyMap.PageUp = keys.PrevPage
	vp.KeyMap.PageDown = keys.NextPage
	vp.KeyMap.Up = keys.CursorUp
	vp.KeyMap.Down = keys.CursorDown

	p := &InfoPage{
		host:     host,
		allLines: lines,
		viewport: vp,
		search:   searchbar.New("Search: "),
		help:     help.New(),
	}
	p.refreshViewport()
	return p
}

func (p *InfoPage) Update(msg tea.Msg) (base.Component, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		// back key: esc clears filter first if active, then navigates back
		if key.Matches(keyMsg, keys.Back) {
			if p.search.Value() != "" {
				p.search.SetValue("")
				p.refreshViewport()
				return p, nil
			}
			return p, func() tea.Msg { return base.OpenHostPageMsg{} }
		} else if key.Matches(keyMsg, keys.ShowFullHelp) {
			p.help.ShowAll = !p.help.ShowAll
			return p, nil
		}
	}

	// Update viewport
	newModel, listCmd := p.viewport.Update(msg)
	p.viewport = newModel
	cmds = append(cmds, listCmd)

	// Update search
	var searchCmd tea.Cmd
	var changed bool
	searchCmd, changed = p.search.Update(msg)
	if changed {
		p.refreshViewport()
	}
	cmds = append(cmds, searchCmd)

	return p, tea.Batch(cmds...)
}

func (p *InfoPage) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.search.SetWidth(w)
	p.viewport.SetWidth(w)
	p.help.SetWidth(w)
}

func (p *InfoPage) View() string {
	title := styles.TitleStyle.Render("SSH Config: " + p.host.Name)
	search := p.search.View()
	helpView := p.help.View(p)

	overhead := lipgloss.Height(title) + lipgloss.Height(search) + lipgloss.Height(helpView)
	vpH := p.height - overhead
	if vpH < 1 {
		vpH = 1
	}
	p.viewport.SetHeight(vpH)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		search,
		p.viewport.View(),
		helpView,
	)
}

// refreshViewport recalcula el contenido del viewport aplicando el filtro actual.
func (p *InfoPage) refreshViewport() {
	q := strings.ToLower(p.search.Value())

	// Índice clave→líneas para agrupar por sección.
	byKey := make(map[string][]parsedLine, len(p.allLines))
	for _, l := range p.allLines {
		byKey[l.key] = append(byKey[l.key], l)
	}

	// Claves ya asignadas a una sección para calcular "Other" después.
	assigned := make(map[string]bool)

	var sb strings.Builder

	renderLine := func(l parsedLine) {
		k := highlightMatchStyled(l.key, q, keyStyle)
		v := highlightMatch(l.value, q)
		sb.WriteString(k + " " + v + "\n")
	}

	matchesFilter := func(l parsedLine) bool {
		if q == "" {
			return true
		}
		return strings.Contains(strings.ToLower(l.key), q) ||
			strings.Contains(strings.ToLower(l.value), q)
	}

	for _, sec := range sshSections {
		var secLines []parsedLine
		for _, k := range sec.keys {
			assigned[k] = true
			for _, l := range byKey[k] {
				if matchesFilter(l) {
					secLines = append(secLines, l)
				}
			}
		}
		if len(secLines) == 0 {
			continue
		}
		sb.WriteString(sectionStyle.Render(sec.title) + "\n")
		for _, l := range secLines {
			renderLine(l)
		}
	}

	// Sección "Other" con las claves no asignadas a ninguna sección.
	var otherLines []parsedLine
	for _, l := range p.allLines {
		if !assigned[l.key] && matchesFilter(l) {
			otherLines = append(otherLines, l)
		}
	}
	if len(otherLines) > 0 {
		sb.WriteString(sectionStyle.Render("⚙️ Other") + "\n")
		for _, l := range otherLines {
			renderLine(l)
		}
	}

	p.viewport.SetContent(sb.String())
	p.viewport.GotoTop()
}

// highlightMatch resalta todas las ocurrencias de q en s (insensible a mayúsculas).
func highlightMatch(s, q string) string {
	return highlightMatchStyled(s, q, lipgloss.NewStyle())
}

// highlightMatchStyled busca q sobre el texto crudo s y aplica:
//   - base a las partes que NO coinciden
//   - matchStyle a las partes que SÍ coinciden
//
// Buscar sobre s crudo (sin ANSI) garantiza que los índices son correctos.
func highlightMatchStyled(s, q string, base lipgloss.Style) string {
	if q == "" {
		return base.Render(s)
	}
	lower := strings.ToLower(s)
	var out strings.Builder
	i := 0
	for {
		idx := strings.Index(lower[i:], q)
		if idx < 0 {
			out.WriteString(base.Render(s[i:]))
			break
		}
		if idx > 0 {
			out.WriteString(base.Render(s[i : i+idx]))
		}
		out.WriteString(matchStyle.Render(s[i+idx : i+idx+len(q)]))
		i += idx + len(q)
	}
	return out.String()
}

// parseSSHG ejecuta «ssh -G <name>» y devuelve las líneas descompuestas.
func parseSSHG(host *ssh.HostConfig) []parsedLine {
	out, err := exec.Command("ssh", "-G", host.Name).Output()
	if err != nil {
		return []parsedLine{{key: "error", value: err.Error()}}
	}
	var lines []parsedLine
	for _, raw := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		parts := strings.SplitN(raw, " ", 2)
		if len(parts) == 2 {
			lines = append(lines, parsedLine{key: parts[0], value: parts[1]})
		} else if raw != "" {
			lines = append(lines, parsedLine{key: raw})
		}
	}
	return lines
}
