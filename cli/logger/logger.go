package logger

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render  // bright cyan
	inputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render  // bright cyan
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render  // light blue
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("83")).Render  // light green
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("221")).Render // yellow
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render // light red
)

type Logger struct{}

func (Logger) Prompt(msg string)  { fmt.Println(promptStyle("[?] " + msg)) }
func (Logger) Input(msg string)   { fmt.Print(inputStyle("[>] " + msg)) }
func (Logger) Info(msg string)    { fmt.Println(infoStyle("[i] " + msg)) }
func (Logger) Success(msg string) { fmt.Println(successStyle("[+] " + msg)) }
func (Logger) Warn(msg string)    { fmt.Println(warnStyle("[!] " + msg)) }
func (Logger) Error(msg string)   { fmt.Println(errorStyle("[-] " + msg)) }

// Styled prefixes for direct use with fmt.Printf in main.go
var (
	InfoPrefix    = infoStyle("[i]")
	WarnPrefix    = warnStyle("[!]")
	ErrorPrefix   = errorStyle("[-]")
	SuccessPrefix = successStyle("[+]")
)
