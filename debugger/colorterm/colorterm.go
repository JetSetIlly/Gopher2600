package colorterm

import (
	"bufio"
	"fmt"
	"gopher2600/debugger"
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/debugger/colorterm/easyterm"
	"os"
	"unicode"
	"unicode/utf8"
)

// ColorTerminal implements debugger UI interface with a basic ANSI terminal
type ColorTerminal struct {
	easyterm.Terminal

	reader         *bufio.Reader
	commandHistory []command
}

type command struct {
	input []byte
}

// Initialise perfoms any setting up required for the terminal
func (ct *ColorTerminal) Initialise() error {
	err := ct.Terminal.Initialise(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	ct.reader = bufio.NewReader(os.Stdin)
	ct.commandHistory = make([]command, 0)

	return nil
}

// CleanUp perfoms any cleaning up required for the terminal
func (ct *ColorTerminal) CleanUp() {
	fmt.Print("\r")
	_ = ct.Flush()
}

// UserPrint implementation for debugger.UI interface
func (ct *ColorTerminal) UserPrint(pp debugger.PrintProfile, s string, a ...interface{}) {
	if pp != debugger.Input {
		fmt.Print("\r")
	}

	switch pp {
	case debugger.CPUStep:
		fmt.Print(ansi.PenColor["yellow"])
	case debugger.VideoStep:
		fmt.Print(ansi.DimPens["yellow"])
	case debugger.MachineInfo:
		fmt.Print(ansi.PenColor["cyan"])
	case debugger.Error:
		fmt.Print(ansi.PenColor["red"])
		fmt.Print(ansi.PenColor["bold"])
		fmt.Print("* ")
		fmt.Print(ansi.NormalPen)
		fmt.Print(ansi.PenColor["red"])
	case debugger.Feedback:
		fmt.Print(ansi.DimPens["white"])
	case debugger.Script:
		fmt.Print("> ")
	case debugger.Prompt:
		fmt.Print(ansi.PenStyles["bold"])
	}

	fmt.Printf(s, a...)
	fmt.Print(ansi.NormalPen)

	// add a newline if print profile is anything other than prompt
	if pp != debugger.Prompt && pp != debugger.Input {
		fmt.Println("")
	}
}

// UserRead implementation for debugger.UI interface
func (ct *ColorTerminal) UserRead(input []byte, prompt string) (int, error) {
	ct.RawMode()
	defer ct.CanonicalMode()

	// er is used to store encoded runes (length of 4 should be enough)
	er := make([]byte, 4)

	n := 0
	cursor := 0
	history := len(ct.commandHistory)

	// buffInput is used to store the latest input when we scroll through
	// history - we don't want to lose what we've typed in case the user wants
	// to resume where we left off
	buffInput := make([]byte, cap(input))
	buffN := 0

	// the method for cursor placement is as follows:
	// 	1. for each iteration in the loop
	//		2. store current cursor position
	//		3. clear the current line
	//		4. output the prompt
	//		5. output the input buffer
	//		6. restore the cursor position
	//
	// for this to work we need to place the cursor in it's initial position,
	fmt.Printf("\r%s", ansi.CursorMove(len(prompt)))

	for {
		fmt.Print(ansi.CursorStore)
		ct.UserPrint(debugger.Prompt, "%s%s", ansi.ClearLine, prompt)
		ct.UserPrint(debugger.Input, string(input[:n]))
		fmt.Print(ansi.CursorRestore)

		r, _, err := ct.reader.ReadRune()
		if err != nil {
			return n, err
		}

		switch r {
		case 3:
			// CTRL-C
			fmt.Print("\n")
			return n + 1, &debugger.UserInterrupt{Message: "user interrupt: CTRL-C"}

		case 13:
			// CARRIAGE RETURN

			// check to see if input is the same as the last history entry
			newEntry := false
			if n > 0 {
				newEntry = true
				if len(ct.commandHistory) > 0 {
					lastHistoryEntry := ct.commandHistory[len(ct.commandHistory)-1].input
					if len(lastHistoryEntry) == n {
						newEntry = false
						for i := 0; i < n; i++ {
							if input[i] != lastHistoryEntry[i] {
								newEntry = true
								break
							}
						}
					}
				}
			}

			// if input is not the same as the last history entry then append a
			// new entry to the history list
			if newEntry {
				nh := make([]byte, n)
				copy(nh, input[:n])
				ct.commandHistory = append(ct.commandHistory, command{input: nh})
			}

			fmt.Print("\n")
			return n + 1, nil

		case 27:
			// ESCAPE SEQUENCE BEGIN
			r, _, err := ct.reader.ReadRune()
			if err != nil {
				return n, err
			}
			switch r {
			case 91:
				// CURSOR KEY
				r, _, err := ct.reader.ReadRune()
				if err != nil {
					return n, err
				}

				switch r {
				case 'A':
					// move up through command history
					if len(ct.commandHistory) > 0 {
						// if we're at the end of the command history then store
						// the current input in buffInput for possible later editing
						if history == len(ct.commandHistory) {
							copy(buffInput, input[:n])
							buffN = n
						}

						if history > 0 {
							history--
							copy(input, ct.commandHistory[history].input)
							n = len(ct.commandHistory[history].input)
							fmt.Print(ansi.CursorMove(n - cursor))
							cursor = n
						}
					}
				case 'B':
					// move down through command history
					if len(ct.commandHistory) > 0 {
						if history < len(ct.commandHistory)-1 {
							history++
							copy(input, ct.commandHistory[history].input)
							n = len(ct.commandHistory[history].input)
							fmt.Print(ansi.CursorMove(n - cursor))
							cursor = n
						} else if history == len(ct.commandHistory)-1 {
							history++
							copy(input, buffInput)
							n = buffN
							fmt.Print(ansi.CursorMove(n - cursor))
							cursor = n
						}
					}
				case 'C':
					// move forward through current command input
					if cursor < n {
						fmt.Print(ansi.CursorForwardOne)
						cursor++
					}
				case 'D':
					// move backward through current command input
					if cursor > 0 {
						fmt.Print(ansi.CursorBackwardOne)
						cursor--
					}

				case 51:
					// DELETE
					if cursor < n {
						copy(input[cursor:], input[cursor+1:])
						cursor--
						n--
						history = len(ct.commandHistory)
					}
				}
			}

		case 127:
			// BACKSPACE
			if cursor > 0 {
				copy(input[cursor-1:], input[cursor:])
				fmt.Print(ansi.CursorBackwardOne)
				cursor--
				n--
				history = len(ct.commandHistory)
			}

		default:
			if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
				fmt.Printf("%c", r)
				m := utf8.EncodeRune(er, r)
				copy(input[cursor+m:], input[cursor:])
				copy(input[cursor:], er[:m])
				cursor++
				n += m
				history = len(ct.commandHistory)
			}
		}

	}
}
