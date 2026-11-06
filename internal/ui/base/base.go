// Package base holds the contracts and messages shared by every UI page:
// the Component interface, navigation/action messages and key bindings.
package base

import tea "charm.land/bubbletea/v2"

// Component is the contract every page and modal implements. Unlike
// tea.Model, Init is optional and Update returns a Component so pages can be
// swapped by the App router.
type Component interface {
	Update(tea.Msg) (Component, tea.Cmd)
	View() string
	SetSize(width int, height int)
}
