package main

import (
	"strings"

	"github.com/mahtues/go-chat/log"
	"github.com/mahtues/go-chat/tui/textinputcmp"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := NewTui()

	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func NewTui() Tui {
	m := Tui{
		textInput: textinputcmp.New(),
	}

	m.textInput.Placeholder = "text"
	m.textInput.Focus()

	m.textInput.CompleteFunc = NewCommands().CompleteFunc

	return m
}

type Tui struct {
	textInput textinputcmp.Model
}

func (m Tui) Init() tea.Cmd {
	return textinput.Blink
}

func (m Tui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.textInput.SetValue("")
			return m, nil
		case tea.KeyCtrlC:
			return m, tea.Quit
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Tui) View() string {
	return m.textInput.View() + "\n"
}

type Commands []string

func NewCommands() Commands {
	users := []string{"alice", "bob", "charlie", "dave", "erica", "eric", "link", "zelda"}
	groups := []string{"games", "work", "foot", "food"}
	options := []string{"send", "info", "on", "off", "only"}

	cmds := make([]string, 0, (len(users)+len(groups))*len(options))

	for _, opt := range options {
		for _, u := range users {
			cmds = append(cmds, opt+":u:"+u)
		}
		for _, g := range groups {
			cmds = append(cmds, opt+":g:"+g)
		}
	}

	return cmds
}

func (c Commands) Suggest(s string) []string {
	matches := []string{}

	for _, k := range c {
		if strings.HasPrefix(k, s) {
			matches = append(matches, k)
		}
	}

	return matches
}

func (c Commands) CompleteFunc(s string) (string, []string) {
	switch opts := c.Suggest(s); len(opts) {
	case 0:
		return s, nil
	case 1:
		return opts[0] + " ", nil
	default:
		rfirst := []rune(opts[0])
		expanded := s

		for l := len([]rune(s)) + 1; l < len(rfirst); l++ {
			first := string(rfirst[:l])

			for i := 1; i < len(opts); i++ {
				if !strings.HasPrefix(opts[i], first) {
					return expanded, opts
				}
			}

			expanded = first
		}

		return expanded, opts
	}
}
