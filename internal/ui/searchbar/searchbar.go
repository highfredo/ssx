// Package searchbar proporciona una barra de búsqueda reutilizable que combina
// un textinput de bubbles con el estilo FilterBarStyle y el icono de lupa.
package searchbar

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/highfredo/ssx/internal/styles"
	"github.com/highfredo/ssx/internal/ui/glyphs"
)

// icon lleva un espacio delante y uno detrás para que el reset ANSI
// no caiga a mitad del glifo de 2 celdas.
const icon = " " + glyphs.Search + " "

// iconVisualWidth es el ancho real en celdas de terminal del icono:
// 1 espacio + 2 celdas (glifo Nerd Font PUA) + 1 espacio de cola.
// lipgloss.Width() cuenta el glifo PUA como 1, por eso usamos esta constante
// manual para los cálculos de layout en lugar de lipgloss.Width(icon).
const iconVisualWidth = 4

// Model encapsula el textinput y se encarga de renderizar la barra completa.
type Model struct {
	input textinput.Model
}

// New crea una barra de búsqueda con el prompt indicado, ya enfocada.
func New(prompt string) *Model {
	fi := textinput.New()
	fi.Prompt = prompt
	fi.Focus()
	return &Model{input: fi}
}

// Update procesa un mensaje de bubbletea, muta el modelo en lugar y devuelve
// el Cmd resultante junto con un booleano que indica si el texto ha cambiado.
func (m *Model) Update(msg tea.Msg) (tea.Cmd, bool) {
	prev := m.input.Value()
	newInput, cmd := m.input.Update(msg)
	m.input = newInput
	return cmd, m.input.Value() != prev
}

// View renderiza la barra completa al ancho total indicado.
// El icono se coloca a la derecha. El input ya tiene el ancho correcto
// gracias a SetWidth, por lo que no se aplica Width() adicional.
func (m *Model) View() string {
	iconStr := lipgloss.NewStyle().Foreground(styles.ColorMuted).Render(icon)
	return styles.FilterBarStyle.Render(m.input.View() + iconStr)
}

// SetWidth ajusta el ancho interno del textinput para que quepa dentro de la
// barra en el ancho total indicado, reservando iconVisualWidth celdas para el icono.
func (m *Model) SetWidth(totalWidth int) {
	promptW := lipgloss.Width(m.input.Prompt)
	inputW := max(0, totalWidth-styles.FilterBarStyle.GetHorizontalFrameSize()-promptW-iconVisualWidth)
	m.input.SetWidth(inputW)
}

// Height devuelve la altura renderizada de la barra (para cálculos de layout).
func (m *Model) Height() int {
	return lipgloss.Height(styles.FilterBarStyle.Render(""))
}

// Value devuelve el texto actual del campo de búsqueda.
func (m *Model) Value() string {
	return m.input.Value()
}

// SetValue establece el texto del campo de búsqueda.
func (m *Model) SetValue(v string) {
	m.input.SetValue(v)
}
