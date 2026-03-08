package base

import tea "charm.land/bubbletea/v2"

type Component interface {
	Update(tea.Msg) (Component, tea.Cmd)
	View() string
	SetSize(width int, height int)
}
