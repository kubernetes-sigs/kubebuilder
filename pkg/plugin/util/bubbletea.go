package util

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// YesNoModel is a Bubbletea model for yes/no prompts
type YesNoModel struct {
	Question string
	Cursor   int
	Selected bool
}

func (m YesNoModel) Init() tea.Cmd {
	return nil
}

func (m YesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "up", "k":
			m.Cursor = 0 // Yes
		case "right", "l", "down", "j":
			m.Cursor = 1 // No
		case "y", "Y":
			m.Cursor = 0
			m.Selected = true
			return m, tea.Quit
		case "n", "N":
			m.Cursor = 1
			m.Selected = true
			return m, tea.Quit
		case "enter", " ":
			m.Selected = true
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

	question := lipgloss.NewStyle().Bold(true).Render(m.Question)
	yes := "[ ] Yes"
	no := "[ ] No"
	
	// Mark selected option
	if m.Cursor == 0 {
		yes = "[x] " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4D8B")).Render("Yes")
	} else {
		no = "[x] " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4D8B")).Render("No")
	}
	
	return fmt.Sprintf("\n  %s\n\n  › %s\n  › %s\n\n  %s\n", 
		question, yes, no,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("← →: select • enter: choose • y/n: quick select"))
}

// BubbleTeaYesNo prompts the user with a yes/no question using Bubbletea UI
func BubbleTeaYesNo(question string) bool {
	model := YesNoModel{
		Question: question,
		Cursor:   0, // Default to Yes
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("%s [y/n] ", question)
		var response string
		fmt.Scanln(&response)
		return response == "y" || response == "Y"
	}

	m, ok := finalModel.(YesNoModel)
	if !ok || !m.Selected {
		return false
	}

	return m.Cursor == 0 // Return true if Yes (index 0) was selected
}
