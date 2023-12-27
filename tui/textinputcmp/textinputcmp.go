package textinputcmp

import (
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func New() Model {
	m := Model{
		Model:        textinput.New(),
		CompleteFunc: func(s string) (string, []string) { return s, nil },
	}

	return m
}

type Model struct {
	textinput.Model
	CompleteFunc func(s string) (string, []string)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyTab {
		if !m.Focused() {
			return m, nil
		}

		rtext, pos := []rune(m.Value()), m.Position()

		left, right := rtext[:pos], string(rtext[pos:])

		completed, _ := m.CompleteFunc(string(left))
		pos = utf8.RuneCountInString(completed)

		// create command for suggestions

		m.SetValue(completed + right)
		m.SetCursor(pos)
		return m, nil
	}

	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}
