package list

import (
	"fmt"
	"os"

	blist "charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ListConfig configures the list component.
type ListConfig struct {
	Title   string
	Options []string
}

// ShowList displays an interactive list and returns the selected option.
// Returns empty string if the operation is cancelled.
func ShowList(config ListConfig) string {
	items := make([]blist.Item, len(config.Options))
	for i, opt := range config.Options {
		items[i] = listItem{title: opt}
	}

	const defaultWidth = 40
	const defaultHeight = 20

	delegate := blist.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := blist.New(items, delegate, defaultWidth, defaultHeight)
	l.Title = config.Title
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	m := model{
		list: l,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(model); ok && m.selected != "" {
		return m.selected
	}

	return ""
}

type listItem struct {
	title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }

type model struct {
	list     blist.Model
	selected string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if item := m.list.SelectedItem(); item != nil {
				if li, ok := item.(listItem); ok {
					m.selected = li.title
				}
			}
			return m, tea.Quit
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	return tea.NewView(m.list.View())
}

// 教程
// package main

// import (
// 	"fmt"
// 	"main/list"
// )

// func main() {
// 	config := list.ListConfig{
// 		Title: "Choose your starter",
// 		Options: []string{
// 			"Bulbasaur",
// 			"Charmander",
// 			"Squirtle",
// 			"Pikachu",
// 		},
// 	}

// 	result := list.ShowList(config)

// 	if result != "" {
// 		fmt.Printf("\n--- You Selected ---\n")
// 		fmt.Printf("%s\n", result)
// 	} else {
// 		fmt.Println("\nOperation cancelled.")
// 	}
// }
