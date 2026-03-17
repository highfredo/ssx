package list

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/searchbar"
)

// Model embeds list.Model so callers can use fl.Title, fl.Styles,
// fl.AdditionalShortHelpKeys, fl.SelectedItem(), etc. directly — identical
// interface to list.Model.
//
// The list is always in Unfiltered or FilterApplied state — never in
// Filtering state — so selection highlighting and the help bar remain
// fully functional.
//
// Model overrides Update, SetSize and View; all other list.Model
// methods/fields are promoted and usable without any prefix.
type Model struct {
	list.Model
	search *searchbar.Model
	width  int
	height int
}

// NewDefaultDelegate creates a DefaultDelegate with the cursor style already applied.
// Use this (or embed it) as the base for any custom delegate passed to New().
func NewDefaultDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	cursorBorder := lipgloss.Border{Left: "❯"}
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		BorderStyle(cursorBorder).
		BorderLeft(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		BorderLeftForeground(styles.ColorPrimary)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		BorderStyle(lipgloss.Border{Left: " "}).
		BorderLeft(true).
		Foreground(lipgloss.Color("#C4B5FD")).
		BorderLeftForeground(styles.ColorPrimary)
	return d
}

// New creates a Model configured for always-active filtering.
func New(items []list.Item, d list.ItemDelegate, w, h int) *Model {
	l := list.New(items, d, w, h)
	l.SetShowStatusBar(false)
	// Title and filter are rendered manually in View(), so hide them from the
	// list's own View to avoid duplication and sizing issues.
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	// Disable list-managed filter keybindings: we manage filtering ourselves.
	l.KeyMap.Filter.SetEnabled(false)
	l.KeyMap.ClearFilter.SetEnabled(false)
	l.KeyMap.CancelWhileFiltering.SetEnabled(false)
	l.KeyMap.AcceptWhileFiltering.SetEnabled(false)

	// Arrow-only navigation avoids conflicts with filter text input.
	l.KeyMap.CursorUp = key.NewBinding(key.WithKeys("up"))
	l.KeyMap.CursorDown = key.NewBinding(key.WithKeys("down"))
	l.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "prev page"))
	l.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "next page"))
	l.KeyMap.GoToStart = key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "go to start"))
	l.KeyMap.GoToEnd = key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "go to end"))
	l.KeyMap.Filter = key.NewBinding()
	l.KeyMap.ClearFilter = key.NewBinding()
	l.KeyMap.ShowFullHelp = key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "more"))
	l.KeyMap.CloseFullHelp = key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "close help"))

	// Apply title style centrally so callers only need to set l.Title.
	l.Styles.Title = styles.TitleStyle

	l.SetShowHelp(false) // rendered manually below the counter in View()

	return &Model{Model: l, search: searchbar.New("Search: ")}
}

// HandleKey intercepts navigation key presses and moves the list cursor
// directly, bypassing the filter input.
// Returns true if the key was consumed. Call this BEFORE your action keys.
func (l *Model) HandleKey(msg tea.KeyPressMsg) bool {
	// Use containsKey instead of key.Matches so navigation works even when
	// updateKeybindings() has temporarily disabled the bindings.
	switch {
	case key.Matches(msg, l.KeyMap.CursorUp):
		l.CursorUp()
		return true
	case key.Matches(msg, l.KeyMap.CursorDown):
		l.CursorDown()
		return true
	case key.Matches(msg, l.KeyMap.PrevPage):
		l.PrevPage()
		return true
	case key.Matches(msg, l.KeyMap.NextPage):
		l.NextPage()
		return true
	case key.Matches(msg, l.KeyMap.GoToStart):
		l.GoToStart()
		return true
	case key.Matches(msg, l.KeyMap.GoToEnd):
		l.GoToEnd()
		return true
	}
	return false
}

// ClearFilter clears the filter text and resets the list to Unfiltered state.
// Call this when the user presses esc on pages where esc means "clear filter".
func (l *Model) ClearFilter() {
	l.search.SetValue("")
	l.ResetFilter()
}

func (l *Model) ShortHelp() []key.Binding {
	var result []key.Binding
	for _, b := range l.Model.ShortHelp() {
		if b.Help().Key != "" {
			result = append(result, b)
		}
	}
	return result
}

func (l *Model) FullHelp() [][]key.Binding {
	var result [][]key.Binding
	for _, group := range l.Model.FullHelp() {
		var filtered []key.Binding
		for _, b := range group {
			if b.Help().Key != "" {
				filtered = append(filtered, b)
			}
		}
		if len(filtered) > 0 {
			result = append(result, filtered)
		}
	}
	return result
}

// Update routes key presses to the search bar, re-filters when the value
// changes, and delegates all remaining handling to the list.
// Navigation keys must be consumed by HandleKey before reaching this method.
func (l *Model) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if handled := l.HandleKey(keyMsg); handled {
			return nil
		}
		// Feed the key to the search bar.
		var searchCmd tea.Cmd
		var changed bool
		searchCmd, changed = l.search.Update(msg)
		cmds = append(cmds, searchCmd)

		if changed {
			l.SetFilterText(l.search.Value())
		}
	}

	// Delegate to the list for selection, mouse, window resize, etc.
	// Use l.Model.Update explicitly to avoid calling ourselves recursively.
	newModel, listCmd := l.Model.Update(msg)
	l.Model = newModel
	cmds = append(cmds, listCmd)

	return tea.Batch(cmds...)
}

// SetSize sets the list dimensions.
func (l *Model) SetSize(w, h int) {
	l.width = w
	l.height = h
	l.search.SetWidth(w)
	l.Model.SetWidth(w)
}

func (l *Model) SearchValue() string {
	return l.search.Value()
}

// View renders: [title] | [search bar] | [list items + pagination] | [counter] | [help].
func (l *Model) View() string {
	title := l.renderTitle()
	search := l.search.View()
	status := l.renderStatus()
	help := l.renderHelp()

	overhead := lipgloss.Height(title) + lipgloss.Height(search) + lipgloss.Height(status) + lipgloss.Height(help)
	listH := l.height - overhead
	if listH < 0 {
		listH = 0
	}
	l.Model.SetHeight(listH)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		search,
		l.Model.View(),
		status,
		help,
	)
}

func (l *Model) renderTitle() string {
	return l.Styles.Title.Render(l.Title)
}

func (l *Model) renderStatus() string {
	total := len(l.Items())
	visible := len(l.VisibleItems())
	filter := l.search.Value()

	var status string
	if filter != "" {
		status = fmt.Sprintf(`%d of %d items`, visible, total)
	} else {
		status = fmt.Sprintf("%d items", total)
	}

	return l.Styles.StatusBar.Render(status)
}

func (l *Model) renderHelp() string {
	return l.Help.View(l)
}
