// Package list wraps bubbles/list with an always-visible search bar, custom
// navigation keys and an optional "virtual" item derived from the query.
package list

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/base"
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
	search     *searchbar.Model
	width      int
	height     int
	virtualFn  func(string) list.Item
	hasVirtual bool
	delegate   list.ItemDelegate
	compact    bool
	statusFn   StatusFunc
	emptyText  string
}

// StatusFunc builds the counter line shown under the list from the visible
// items, the total and the current search query.
type StatusFunc func(visible []list.Item, total int, query string) string

// Compactable is implemented by delegates that can render a denser row when
// the terminal is narrow (see NarrowWidth).
type Compactable interface {
	Compact() list.ItemDelegate
}

// NarrowWidth is the terminal width below which lists switch to compact
// rendering: one-line rows and short help only.
const NarrowWidth = 70

// NewDefaultDelegate creates a DefaultDelegate with the cursor style already applied.
// Use this (or embed it) as the base for any custom delegate passed to New().
func NewDefaultDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	cursorBorder := lipgloss.Border{Left: "❯"}
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		BorderStyle(cursorBorder).
		BorderLeft(true).
		Foreground(styles.ColorText).
		BorderLeftForeground(styles.ColorPrimary)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.
		BorderStyle(lipgloss.Border{Left: " "}).
		BorderLeft(true).
		Foreground(styles.ColorSecondary).
		BorderLeftForeground(styles.ColorPrimary)
	d.Styles.DimmedTitle = d.Styles.DimmedTitle.Foreground(styles.ColorMuted)
	d.Styles.DimmedDesc = d.Styles.DimmedDesc.Foreground(styles.ColorMuted)
	return d
}

// CompactDelegate returns d without description line or spacing, for narrow
// terminals.
func CompactDelegate(d list.DefaultDelegate) list.DefaultDelegate {
	d.ShowDescription = false
	d.SetSpacing(0)
	return d
}

// TruncateLine cuts a rendered line to width with an ellipsis, preserving
// ANSI styling. Use it after appending decorations to a delegate row.
func TruncateLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	return ansi.Truncate(line, width, "…")
}

// New creates a Model configured for always-active filtering. The list has
// no size until SetSize is called, normally on the first tea.WindowSizeMsg.
func New(items []list.Item, d list.ItemDelegate, keys base.NavigationKeys) *Model {

	l := list.New(items, d, 0, 0)
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
	l.KeyMap.CursorUp = keys.CursorUp
	l.KeyMap.CursorDown = keys.CursorDown
	l.KeyMap.PrevPage = keys.PrevPage
	l.KeyMap.NextPage = keys.NextPage
	l.KeyMap.GoToStart = keys.GoToStart
	l.KeyMap.GoToEnd = keys.GoToEnd
	l.KeyMap.Filter = key.NewBinding()
	l.KeyMap.ClearFilter = key.NewBinding()
	l.KeyMap.ShowFullHelp = keys.ShowFullHelp
	l.KeyMap.CloseFullHelp = keys.CloseFullHelp

	// Apply title style centrally so callers only need to set l.Title.
	l.Styles.Title = styles.TitleStyle

	l.SetShowHelp(false) // rendered manually below the counter in View()

	m := &Model{
		Model:     l,
		search:    searchbar.New("Search: "),
		delegate:  d,
		statusFn:  defaultStatus,
		emptyText: "Nothing to show",
	}
	m.Filter = func(term string, targets []string) []list.Rank {
		if !m.hasVirtual || len(targets) == 0 {
			return Filter(term, targets)
		}

		realTargets := targets[:len(targets)-1]
		ranks := Filter(term, realTargets)
		ranks = append(ranks, list.Rank{
			Index:          len(targets) - 1,
			MatchedIndexes: []int{},
		})

		return ranks
	}

	return m
}

// Filter wraps list.DefaultFilter converting MatchedIndexes from byte
// offsets (as returned by sahilm/fuzzy) to rune offsets, which is what
// lipgloss.StyleRunes expects when highlighting. Without this, any multibyte
// rune in the target (e.g. "→") shifts the highlight of later characters.
// Use it anywhere matched runes are going to be styled.
func Filter(term string, targets []string) []list.Rank {
	ranks := list.DefaultFilter(term, targets)
	for i, r := range ranks {
		ranks[i].MatchedIndexes = byteToRuneIndexes(targets[r.Index], r.MatchedIndexes)
	}
	return ranks
}

func byteToRuneIndexes(s string, byteIdx []int) []int {
	if len(byteIdx) == 0 {
		return byteIdx
	}
	runeAt := make(map[int]int, len(s))
	n := 0
	for b := range s {
		runeAt[b] = n
		n++
	}
	out := make([]int, 0, len(byteIdx))
	for _, b := range byteIdx {
		if r, ok := runeAt[b]; ok {
			out = append(out, r)
		}
	}
	return out
}

// SetVirtualItemBuilder configures a callback invoked whenever the search query changes.
// Returning a non-nil item appends/updates a virtual item at the end of the list;
// returning nil removes the virtual item if present.
func (l *Model) SetVirtualItemBuilder(fn func(string) list.Item) {
	l.virtualFn = fn
}

// VirtualItem returns the current virtual item and its index in the list, or (nil, -1).
func (l *Model) VirtualItem() (list.Item, int) {
	if !l.hasVirtual {
		return nil, -1
	}
	items := l.Items()
	if len(items) == 0 {
		return nil, -1
	}
	return items[len(items)-1], len(items) - 1
}

// PromoteVirtualItem converts the current virtual item into a permanent list
// item. The search is cleared so the full list is shown again with the new
// item selected at the end.
func (l *Model) PromoteVirtualItem(it list.Item) tea.Cmd {
	if !l.hasVirtual {
		return nil
	}
	l.hasVirtual = false
	idx := len(l.Items()) - 1
	cmd := l.SetItem(idx, it)
	l.search.SetValue("")
	l.ResetFilter()
	l.Select(idx)
	return cmd
}

// HandleKey intercepts navigation key presses and moves the list cursor
// directly, bypassing the filter input.
// Returns true if the key was consumed. Call this BEFORE your action keys.
func (l *Model) HandleKey(msg tea.KeyPressMsg) bool {
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
	case key.Matches(msg, l.KeyMap.ShowFullHelp), key.Matches(msg, l.KeyMap.CloseFullHelp):
		l.Help.ShowAll = !l.Help.ShowAll
		return true
	}
	return false
}

// ClearFilter clears the filter text and resets the list to Unfiltered state.
// Call this when the user presses esc on pages where esc means "clear filter".
func (l *Model) ClearFilter() {
	if l.hasVirtual {
		l.hasVirtual = false
		l.RemoveItem(len(l.Items()) - 1)
	}
	l.search.SetValue("")
	l.ResetFilter()
}

// ShortHelp implements help.KeyMap, hiding bindings without help text.
func (l *Model) ShortHelp() []key.Binding {
	var result []key.Binding
	for _, b := range l.Model.ShortHelp() {
		if b.Help().Key != "" {
			result = append(result, b)
		}
	}
	return result
}

// FullHelp implements help.KeyMap, hiding bindings without help text.
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

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if l.HandleKey(msg) {
			return nil
		}
		cmds = append(cmds, l.feedSearch(msg))
	case tea.PasteMsg:
		cmds = append(cmds, l.feedSearch(msg))
	}

	// Delegate to the list for selection, mouse, window resize, etc.
	// Use l.Model.Update explicitly to avoid calling ourselves recursively.
	newModel, listCmd := l.Model.Update(msg)
	l.Model = newModel
	cmds = append(cmds, listCmd)

	return tea.Batch(cmds...)
}

// feedSearch forwards msg to the search bar and re-filters when its value
// changes.
func (l *Model) feedSearch(msg tea.Msg) tea.Cmd {
	cmd, changed := l.search.Update(msg)
	if changed {
		l.applyFilter()
	}
	return cmd
}

func (l *Model) applyFilter() {
	l.syncVirtualItem()
	val := l.search.Value()
	if val == "" {
		l.ResetFilter()
		return
	}
	l.SetFilterText(val)
}

func (l *Model) syncVirtualItem() {
	if l.virtualFn == nil {
		return
	}
	itemsLen := len(l.Items())
	if vItem := l.virtualFn(l.search.Value()); vItem != nil {
		if l.hasVirtual {
			_ = l.SetItem(itemsLen-1, vItem)
			return
		}
		l.hasVirtual = true
		_ = l.InsertItem(itemsLen, vItem)
	} else if l.hasVirtual {
		l.hasVirtual = false
		l.RemoveItem(itemsLen - 1)
	}
}

// SetStatusFunc replaces the default "N items" counter.
func (l *Model) SetStatusFunc(fn StatusFunc) {
	if fn != nil {
		l.statusFn = fn
	}
}

// SetEmptyText sets the message shown when the list has no items at all.
func (l *Model) SetEmptyText(text string) {
	l.emptyText = text
}

// SetSize sets the list dimensions and switches to compact rendering when
// the width is below NarrowWidth.
func (l *Model) SetSize(w, h int) {
	l.width = w
	l.height = h
	l.search.SetWidth(w)
	l.Model.SetWidth(w)

	compact := w < NarrowWidth
	if compact == l.compact {
		return
	}
	l.compact = compact
	if compact {
		l.Help.ShowAll = false
	}
	if c, ok := l.delegate.(Compactable); ok {
		if compact {
			l.SetDelegate(c.Compact())
		} else {
			l.SetDelegate(l.delegate)
		}
	}
}

// Compact reports whether the list is rendering in narrow mode.
func (l *Model) Compact() bool {
	return l.compact
}

// SearchValue returns the current search query.
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
	listH := max(l.height-overhead, 0)
	l.Model.SetHeight(listH)

	body := l.Model.View()
	if len(l.VisibleItems()) == 0 {
		body = l.renderEmpty(listH)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		search,
		body,
		status,
		help,
	)
}

func (l *Model) renderTitle() string {
	return l.Styles.Title.Render(l.Title)
}

func (l *Model) renderStatus() string {
	// The virtual item is a query artefact, not a real entry: keep it out of
	// the counts. Items are plain comparable values, so == identifies it.
	visible, total := l.VisibleItems(), len(l.Items())
	if v, idx := l.VirtualItem(); idx >= 0 {
		total--
		visible = slices.DeleteFunc(slices.Clone(visible), func(it list.Item) bool { return it == v })
	}
	status := l.statusFn(visible, total, l.search.Value())
	return l.Styles.StatusBar.Render(status)
}

// renderEmpty fills the list area with a hint when nothing is visible.
func (l *Model) renderEmpty(height int) string {
	var msg string
	if q := l.search.Value(); q != "" {
		msg = fmt.Sprintf("No matches for %q  ·  esc to clear", q)
	} else {
		msg = l.emptyText
	}
	return lipgloss.NewStyle().
		Height(height).
		Padding(1, 2).
		Foreground(styles.ColorMuted).
		Render(msg)
}

func defaultStatus(visible []list.Item, total int, query string) string {
	if query != "" {
		return fmt.Sprintf("%d of %d items", len(visible), total)
	}
	return fmt.Sprintf("%d items", total)
}

// renderHelp draws the help line. The short view is hard-truncated because
// bubbles/help only elides items when there is room left for the ellipsis.
func (l *Model) renderHelp() string {
	v := l.Help.View(l)
	if l.Help.ShowAll || l.width <= 0 {
		return v
	}
	return TruncateLine(v, l.width)
}
