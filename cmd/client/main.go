package main

import (
	"fmt"

	"github.com/c-bata/go-prompt"

	customPrompt "github.com/smanhack/gophkeeper/internal/client/prompt"
)

var (
	BuildVersion string
	BuildDate    string
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\n", BuildVersion, BuildDate)

	p := prompt.New(
		customPrompt.NewExecutor().Execute,
		customPrompt.NewCompleter().Complete,
		prompt.OptionTitle("Gophkeeper"),
		prompt.OptionPrefix(">>>"),
		prompt.OptionInputTextColor(prompt.Yellow),
	)
	p.Run()
}
