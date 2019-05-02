package console

import (
	"fmt"
	"gopher2600/gui"
	"io"
	"os"
)

// PlainTerminal is the default, most basic terminal interface
type PlainTerminal struct {
	read  io.Reader
	write io.Writer
}

// Initialise perfoms any setting up required for the terminal
func (pt *PlainTerminal) Initialise() error {
	pt.read = os.Stdin
	pt.write = os.Stdout
	return nil
}

// CleanUp perfoms any cleaning up required for the terminal
func (pt *PlainTerminal) CleanUp() {
}

// RegisterTabCompleter adds an implementation of TabCompleter to the terminal
func (pt *PlainTerminal) RegisterTabCompleter(TabCompleter) {
}

// UserPrint is the plain terminal print routine
func (pt PlainTerminal) UserPrint(pp PrintProfile, s string, a ...interface{}) {
	switch pp {
	case Error:
		s = fmt.Sprintf("* %s", s)
	case Help:
		s = fmt.Sprintf("  %s", s)
	}

	s = fmt.Sprintf(s, a...)
	pt.write.Write([]byte(s))

	if pp != Prompt {
		pt.write.Write([]byte("\n"))
	}
}

// UserRead is the plain terminal read routine
func (pt PlainTerminal) UserRead(input []byte, prompt string, _ chan gui.Event, _ func(gui.Event) error) (int, error) {
	pt.UserPrint(Prompt, prompt)

	n, err := pt.read.Read(input)
	if err != nil {
		return n, err
	}
	return n, nil
}

// IsInteractive satisfies the console.UserInput interface
func (pt *PlainTerminal) IsInteractive() bool {
	return true
}
