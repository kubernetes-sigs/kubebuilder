package util

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define styles
var (
	questionStyle = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4D8B"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
)

// YesNoModel is a Bubbletea model for yes/no prompts
type YesNoModel struct {
	Question string
	Cursor   int
	Options  []string
	Selected bool
	Result   bool
}

func (m YesNoModel) Init() tea.Cmd {
	return nil
}

func (m YesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "left", "h":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j", "right", "l":
			if m.Cursor < len(m.Options)-1 {
				m.Cursor++
			}
		case "enter", " ":
			m.Selected = true
			m.Result = m.Cursor == 0 // Yes is at index 0
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m YesNoModel) View() string {
	if m.Selected {
		return ""
	}

	s := "\n  " + questionStyle.Render(m.Question) + "\n\n"

	for i, option := range m.Options {
		cursor := " "
		style := lipgloss.NewStyle()

		if m.Cursor == i {
			cursor = "›"
			style = selectedStyle

			// Add checkbox for visual feedback
			option = fmt.Sprintf("[x] %s", option)
		} else {
			option = fmt.Sprintf("[ ] %s", option)
		}

		s += fmt.Sprintf("  %s %s\n", cursor, style.Render(option))
	}

	s += "\n  " + helpStyle.Render("j/k: up/down • enter: select • q: quit") + "\n"

	return s
}

// BubbleTeaYesNo prompts the user with a yes/no question using Bubbletea UI
func BubbleTeaYesNo(question string) bool {
	model := YesNoModel{
		Question: question,
		Options:  []string{"Yes", "No"},
		Cursor:   0,
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		// Fallback to simple prompt if Bubbletea fails
		fmt.Printf("%s [y/n] ", question)
		var response string
		fmt.Scanln(&response)
		return strings.ToLower(response) == "y"
	}

	m, ok := finalModel.(YesNoModel)
	if !ok {
		return false
	}

	// If the user quit without selecting, default to false
	if !m.Selected {
		return false
	}

	return m.Result
}
