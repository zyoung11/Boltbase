package textarea

import (
	"log"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TextAreaInput(header string) string {
	p := tea.NewProgram(initialModel(header))
	m, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}
	if finalModel, ok := m.(model); ok {
		return finalModel.textarea.Value()
	}
	return ""
}

type (
	errMsg error
)

type model struct {
	textarea textarea.Model
	header   string
	err      error
	quitting bool
}

func initialModel(header string) model {
	ti := textarea.New()
	ti.Placeholder = "Once upon a time..."
	ti.SetVirtualCursor(false)
	ti.SetStyles(textarea.DefaultStyles(true))
	ti.Focus()

	return model{
		textarea: ti,
		header:   header,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.RequestBackgroundColor)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.textarea.SetStyles(textarea.DefaultStyles(msg.IsDark()))

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) headerView() string {
	return "\n" + m.header + "\n"
}

func (m model) footerView() string {
	return "\n(esc to quit)\n"
}

func (m model) View() tea.View {
	var c *tea.Cursor
	if !m.textarea.VirtualCursor() {
		c = m.textarea.Cursor()

		c.Y += lipgloss.Height(m.headerView())
	}

	str := lipgloss.JoinVertical(lipgloss.Top, m.headerView(), m.textarea.View(), m.footerView())

	if m.quitting {
		str += "\n"
	}

	v := tea.NewView(str)
	v.Cursor = c
	return v
}

// 教程
// package main
// import (
// 	"fmt"
// 	"main/text"
// )
// func main() {
// 	// 调用库函数获取多行输入
// 	input := text.TextAreaInput("请输入你的故事：")
// 	fmt.Println("你输入的内容是：")
// 	fmt.Println(input)
// }
