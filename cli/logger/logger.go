package logger

import "fmt"

type Logger struct{}

func (Logger) Prompt(msg string)  { fmt.Println("[?] " + msg) }
func (Logger) Input(msg string)   { fmt.Print("[>] " + msg) }
func (Logger) Info(msg string)    { fmt.Println("[i] " + msg) }
func (Logger) Success(msg string) { fmt.Println("[+] " + msg) }
func (Logger) Warn(msg string)    { fmt.Println("[!] " + msg) }
func (Logger) Error(msg string)   { fmt.Println("[-] " + msg) }
