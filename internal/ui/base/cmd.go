package base

import tea "charm.land/bubbletea/v2"

// Emit returns a Cmd that delivers msg unchanged. It replaces the common
// `func() tea.Msg { return msg }` closure used to route messages to the App.
func Emit(msg tea.Msg) tea.Cmd {
	return func() tea.Msg { return msg }
}
