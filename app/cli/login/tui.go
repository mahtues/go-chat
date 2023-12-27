package login

import (
	"github.com/mahtues/go-chat/misc"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	username textinput.Model
	password textinput.Model
	next     tea.Model
}

func New(next tea.Model) Model {
	m := Model{
		username: textinput.New(),
		password: textinput.New(),
		next:     next,
	}

	m.username.Placeholder = "username"
	m.username.Focus()

	m.password.Placeholder = "password"
	m.password.EchoMode = textinput.EchoPassword

	return m
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiuCmd tea.Cmd
		tipCmd tea.Cmd
	)

	m.username, tiuCmd = m.username.Update(msg)
	m.password, tipCmd = m.password.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit
		case msg.Type == tea.KeyEnter && m.username.Focused():
			m.username.Blur()
			m.password.Focus()
		case msg.Type == tea.KeyEnter && m.password.Focused():
			if m.username.Value() == m.password.Value() {
				return m.next, nil
			}

			return m, tea.Quit
		}
	}

	return m, tea.Batch(tiuCmd, tipCmd)
}

func (m Model) View() string {
	return misc.Concat("\nlogin\n\n", m.username.View(), "\n\n", m.password.View(), "\n\n")
}
