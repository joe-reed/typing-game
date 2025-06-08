package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"os"
)

type model struct {
	currentWord   string
	position      int
	errorPosition *int
	cursor        cursor.Model
}

func initialModel() model {
	c := cursor.New()
	c.Focus()

	return model{
		currentWord:   "Hello world, this is a typing test! How fast can you type? ",
		position:      0,
		cursor:        c,
		errorPosition: nil,
	}
}

func (m model) Init() tea.Cmd {
	return cursor.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	oldPosition := m.position

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "backspace":
			if m.position != 0 {
				m.position--
			}

			if m.errorPosition != nil && m.position <= *m.errorPosition {
				m.errorPosition = nil
			}
		default:
			if m.position == len(m.currentWord)-1 {
				break
			}

			if msg.String() != string(m.currentWord[m.position]) {
				if m.errorPosition == nil {
					m.errorPosition = new(int)
					*m.errorPosition = m.position
				}
			}

			m.position++
		}

	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.cursor, cmd = m.cursor.Update(msg)
	cmds = append(cmds, cmd)

	if m.position != oldPosition && m.cursor.Mode() == cursor.CursorBlink {
		m.cursor.Blink = false
		cmds = append(cmds, m.cursor.BlinkCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	completedStyle := lipgloss.NewStyle().Background(lipgloss.Color("#2E8B57")).Foreground(lipgloss.Color("#FFF"))
	errorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FF6347")).Foreground(lipgloss.Color("#FFF"))

	m.cursor.SetChar(string(m.currentWord[m.position]))

	var s string

	if m.errorPosition != nil {
		s += completedStyle.Render(m.currentWord[:*m.errorPosition]) +
			errorStyle.Render(m.currentWord[*m.errorPosition:m.position])
	} else {
		s += completedStyle.Render(m.currentWord[:m.position])
	}

	s += m.cursor.View() + m.currentWord[m.position+1:]

	s += "\nPress esc to quit.\n"

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
