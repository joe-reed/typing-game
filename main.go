package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"time"

	"os"
)

type model struct {
	currentWord   string
	position      int
	errorPosition *int
	cursor        cursor.Model
	stopwatch     stopwatch.Model
	times         []time.Duration
}

func (m model) isAtEnd() bool {
	return m.position == len(m.currentWord)
}

func (m model) isCompleted() bool {
	return m.isAtEnd() && !m.hasError()
}

func (m model) hasError() bool {
	return m.errorPosition != nil
}

func (m model) wordsPerMinute() float64 {
	wordCount := len(strings.Split(m.currentWord, " "))
	minutes := m.stopwatch.Elapsed().Minutes()

	if minutes == 0 {
		return 0
	}

	return float64(wordCount) / minutes
}

func initialModel() model {
	c := cursor.New()
	c.Focus()

	return model{
		currentWord:   "Hello world, this is a typing test! How fast can you type?",
		position:      0,
		cursor:        c,
		errorPosition: nil,
		stopwatch:     stopwatch.NewWithInterval(time.Millisecond),
	}
}

func (m model) Init() tea.Cmd {
	return cursor.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	oldPosition := m.position

	m.cursor, cmd = m.cursor.Update(msg)
	cmds = append(cmds, cmd)

	m.stopwatch, cmd = m.stopwatch.Update(msg)
	cmds = append(cmds, cmd)

	if m.isCompleted() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter":
				newModel := initialModel()
				newModel.times = m.times

				return newModel, tea.Batch(cmds...)
			}
		}

		if m.stopwatch.Running() {
			m.times = append(m.times, m.stopwatch.Elapsed())
			return m, m.stopwatch.Stop()
		}

		return m, tea.Batch(cmds...)
	}

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
			if m.isAtEnd() {
				break
			}

			if !m.stopwatch.Running() {
				cmds = append(cmds, m.stopwatch.Start())
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

	if m.position != oldPosition && m.cursor.Mode() == cursor.CursorBlink {
		m.cursor.Blink = false
		cmds = append(cmds, m.cursor.BlinkCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	s := m.renderCurrentWord()
	s += "\n\n"
	s += m.stopwatch.View()
	s += "\n"
	s += fmt.Sprintf("%.2fwpm", m.wordsPerMinute())
	s += "\n"

	for i, t := range m.times {
		s += fmt.Sprintf("Time %d: %s\n", i+1, t.String())
	}

	s += "\n"

	if m.isCompleted() {
		s += "Press enter to restart.\n"
	}

	s += "Press esc to quit.\n"

	return s
}

func (m model) renderCurrentWord() string {
	completedStyle := lipgloss.NewStyle().Background(lipgloss.Color("#2E8B57")).Foreground(lipgloss.Color("#FFF"))
	errorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FF6347")).Foreground(lipgloss.Color("#FFF"))

	var s string

	if m.errorPosition != nil {
		s += completedStyle.Render(m.currentWord[:*m.errorPosition]) +
			errorStyle.Render(m.currentWord[*m.errorPosition:m.position])
	} else {
		s += completedStyle.Render(m.currentWord[:m.position])
	}

	if m.isAtEnd() {
		return s
	}

	m.cursor.SetChar(string(m.currentWord[m.position]))

	s += m.cursor.View() + m.currentWord[m.position+1:]

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
