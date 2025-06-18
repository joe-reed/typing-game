package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wordgen/wordlists/eff"
	"math"
	"math/rand"
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
	highscore     time.Duration
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

func (m model) wordsPerMinute() int64 {
	wordCount := len(strings.Split(m.currentWord, " "))
	seconds := m.stopwatch.Elapsed().Seconds()

	if seconds == 0 {
		return 0
	}

	return int64(math.Trunc(float64(wordCount) / math.Round(seconds) * 60))
}

func initialModel() model {
	c := cursor.New()
	c.Focus()

	return model{
		currentWord:   getSentence(),
		position:      0,
		cursor:        c,
		errorPosition: nil,
		stopwatch:     stopwatch.NewWithInterval(time.Millisecond),
		highscore:     0,
	}
}

func getSentence() string {
	var words []string
	for i := 0; i < 8; i++ {
		idx := rand.Intn(len(eff.Large))
		words = append(words, eff.Large[idx])
	}
	return strings.Join(words, " ")
}

func (m model) Init() tea.Cmd {
	return tea.Batch(cursor.Blink, LoadHighscore)
}

func SaveHighScore(time time.Duration) tea.Cmd {
	return func() tea.Msg {
		file, err := os.OpenFile("highscore.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return tea.Quit()
		}

		defer file.Close()

		_, err = file.WriteString(time.String())

		if err != nil {
			return tea.Quit()
		}

		return nil
	}
}

type LoadedHighscoreMsg struct {
	Highscore time.Duration
}

type FailedToLoadHighscoreMsg struct {
	Error error
}

func LoadHighscore() tea.Msg {
	data, err := os.ReadFile("highscore.txt")

	if err != nil {
		if os.IsNotExist(err) {
			return LoadedHighscoreMsg{Highscore: 0}
		}

		return FailedToLoadHighscoreMsg{Error: err}
	}

	duration, err := time.ParseDuration(strings.TrimSpace(string(data)))
	if err != nil {
		return FailedToLoadHighscoreMsg{Error: err}
	}

	return LoadedHighscoreMsg{Highscore: duration}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(LoadedHighscoreMsg); ok {
		m.highscore = msg.(LoadedHighscoreMsg).Highscore
		return m, nil
	}

	if _, ok := msg.(FailedToLoadHighscoreMsg); ok {
		return m, tea.Quit
	}

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
				newModel.highscore = m.highscore

				return newModel, tea.Batch(cmds...)
			}
		}

		if m.stopwatch.Running() {
			time := m.stopwatch.Elapsed()
			m.times = append(m.times, time)

			if time < m.highscore || m.highscore == 0 {
				m.highscore = time
				cmds = append(cmds, SaveHighScore(m.highscore))
			}

			cmds = append(cmds, m.stopwatch.Stop())

			return m, tea.Batch(cmds...)
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
	s += fmt.Sprintf("%dwpm", m.wordsPerMinute())
	s += "\n"

	for i, t := range m.times {
		s += fmt.Sprintf("Time %d: %s\n", i+1, t.String())
	}

	s += "\n"

	s += fmt.Sprintf("Highscore: %s\n", m.highscore.String())

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
