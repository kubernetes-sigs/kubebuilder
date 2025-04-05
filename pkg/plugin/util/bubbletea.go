package util

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Color constants
const (
	HighlightColor = "#FF4D8B"
	HelpTextColor  = "#626262"
)

// YesNoModel represents a simple yes/no selection prompt.
// It implements the tea.Model interface for use with Bubble Tea.
type YesNoModel struct {
	// Question is the prompt text shown to the user
	Question string
	// Choice tracks the selection (true = Yes, false = No)
	Choice bool
	// Done indicates whether the selection has been confirmed
	Done bool
}

// Init fulfills the tea.Model interface but doesn't need to do anything
// for this simple model.
func (m YesNoModel) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the model state accordingly.
// It processes keystrokes to navigate between options and confirm selections.
func (m YesNoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "left", "up":
			m.Choice = true
		case "n", "N", "right", "down":
			m.Choice = false
		case "enter", " ":
			m.Done = true
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the current state of the model as a string.
// When done, it shows the final selection instead of clearing the screen.
func (m YesNoModel) View() string {
	if m.Done {
		result := "No"
		if m.Choice {
			result = "Yes"
		}
		return fmt.Sprintf("%s: %s", m.Question, result)
	}

	// Style definitions
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(HighlightColor))
	boldStyle := lipgloss.NewStyle().Bold(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(HelpTextColor))

	// Prepare options based on current selection
	yes := "[ ] Yes"
	no := "[ ] No"
	if m.Choice {
		yes = "[x] " + highlightStyle.Render("Yes")
	} else {
		no = "[x] " + highlightStyle.Render("No")
	}

	// Render the full prompt
	return fmt.Sprintf("\n  %s\n\n  › %s\n  › %s\n\n  %s\n",
		boldStyle.Render(m.Question),
		yes, no,
		helpStyle.Render("y/n: select • enter: confirm"))
}

// BubbleTeaYesNo presents a user with a yes/no prompt using Bubble Tea UI.
//
// It displays an interactive prompt with the given question and returns
// the user's selection as a boolean (true = Yes, false = No).
//
// If Bubble Tea encounters an error, it falls back to a simple terminal prompt.
//
// Example usage:
//
//	if util.BubbleTeaYesNo("Create Resource?") {
//	    // User selected "Yes"
//	} else {
//	    // User selected "No"
//	}
func BubbleTeaYesNo(question string) bool {
	model := YesNoModel{
		Question: question,
		Choice:   true, // Default to Yes
	}

    p := tea.NewProgram(model)
	finalModel, err := p.Run()

	if err != nil {
		// Fallback to simple prompt if error
		fmt.Printf("%s [y/n] ", question)
		var response string
		fmt.Scanln(&response)
		return response == "y" || response == "Y"
	}

	m, ok := finalModel.(YesNoModel)
	if !ok || !m.Done {
		// User quit without confirming - print the result
		fmt.Printf("%s: No (cancelled)\n", question)
		return false
	}

	// User made a selection - print the result
	result := "No"
	if m.Choice {
		result = "Yes"
	}
	fmt.Printf("%s: %s\n", question, result)

	return m.Choice
}
