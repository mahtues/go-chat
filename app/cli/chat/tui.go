package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mahtues/go-chat/data"
)

type Model struct {
	viewport  viewport.Model
	textinput textinput.Model
	text      string
	err       error

	SendTextTo func(data.Event)
	id         uint64
}

func New() Model {
	m := Model{
		viewport:  viewport.New(120, 10),
		textinput: textinput.New(),
	}

	m.viewport.KeyMap = viewport.KeyMap{
		Up: key.NewBinding(
			key.WithKeys("ctrl+k"),
		),
		Down: key.NewBinding(
			key.WithKeys("ctrl+j"),
		),
	}

	m.textinput.Focus()
	m.textinput.Placeholder = "message..."

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		vpCmd tea.Cmd
		tiCmd tea.Cmd
	)

	m.viewport, vpCmd = m.viewport.Update(msg)
	m.textinput, tiCmd = m.textinput.Update(msg)

	add := func(s string) {
		if m.text == "" {
			m.text = s
		} else {
			m.text = strings.Join([]string{m.text, "\n", s}, "")
		}
	}

	atBottom := m.viewport.AtBottom()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			input := m.textinput.Value()
			if input == "" {
				break
			}
			m.SendTextTo(data.Event{Id: m.id, TextTo: &data.TextTo{"guest", input}})
			m.viewport.SetContent(m.text)
			if atBottom {
				m.viewport.GotoBottom()
			}
			m.textinput.Reset()
			m.id++
		case tea.KeyCtrlG:
			m.viewport.GotoBottom()
		}
	case data.Event:
		add(fmt.Sprintf("%s: %s", msg.TextTo.To, msg.TextTo.Text))
		m.viewport.SetContent(m.text)
		if atBottom {
			m.viewport.GotoBottom()
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) View() string {
	return fmt.Sprintf("\nchat with guest\n\n%s\n\n%s\n\n", m.viewport.View(), m.textinput.View())
}
