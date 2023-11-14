package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TuiRun() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
	log.Fatal("exit")
}

type model struct {
	viewport    viewport.Model
	messages    strings.Builder
	textarea    textarea.Model
	senderStyle lipgloss.Style
	err         error
}

func initialModel() *model {
	ta := textarea.New()
	ta.Placeholder = "type message..."
	ta.Focus()
	ta.Prompt = "\u2502 "
	ta.CharLimit = 280
	ta.SetWidth(120)
	ta.SetHeight(10)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	//ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 5)
	vp.SetContent("welcome...")

	return &model{
		textarea:    ta,
		messages:    strings.Builder{},
		viewport:    vp,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
		err:         nil,
	}
}

func (m *model) Init() tea.Cmd {
	return textarea.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			fmt.Println(m.textarea.Value())
			return m, tea.Quit
		case tea.KeyCtrlD:
			m.messages.WriteString(m.senderStyle.Render("You: ") + strings.Join(strings.Split(m.textarea.Value(), "\n"), "\nYou: "))
			m.viewport.SetContent(m.messages.String())
			m.textarea.Reset()
			m.viewport.GotoBottom()
		}
	case error:
		m.err = msg
		return m, nil
	}
	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *model) View() string {
	return fmt.Sprintf("\n%s\n\n%s\n\n", m.viewport.View(), m.textarea.View())
}
