package colorterm

import (
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/debugger/colorterm/easyterm"
	"gopher2600/debugger/ui"
	"unicode"
	"unicode/utf8"
)

// UserRead is the top level input function
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
	ct.Print("\r%s", ansi.CursorMove(len(prompt)))

	for {
		ct.Print(ansi.CursorStore)
		ct.UserPrint(ui.Prompt, "%s%s", ansi.ClearLine, prompt)
		ct.UserPrint(ui.Input, string(input[:n]))
		ct.Print(ansi.CursorRestore)

		r, _, err := ct.reader.ReadRune()
		if err != nil {
			return n, err
		}

		switch r {
		case easyterm.KeyTab:
			if ct.tabCompleter != nil {
				s := ct.tabCompleter.GuessWord(string(input[:cursor]))

				// the difference in the length of the new input and the old
				// input
				d := len(s) - cursor

				// append everythin after the cursor to the new string and copy
				// into input array
				s += string(input[cursor:])
				copy(input, []byte(s))

				// advance character to end of completed word
				ct.Print(ansi.CursorMove(d))
				cursor += d

				// note new used-length of input array
				n += d
			}

		case easyterm.KeyCtrlC:
			// CTRL-C
			ct.Print("\n")
			return n + 1, &ui.UserInterrupt{Message: "user interrupt: CTRL-C"}

		case easyterm.KeyCarriageReturn:
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

			ct.Print("\n")
			return n + 1, nil

		case easyterm.KeyEsc:
			// ESCAPE SEQUENCE BEGIN
			r, _, err := ct.reader.ReadRune()
			if err != nil {
				return n, err
			}
			switch r {
			case easyterm.EscCursor:
				// CURSOR KEY
				r, _, err := ct.reader.ReadRune()
				if err != nil {
					return n, err
				}

				switch r {
				case easyterm.CursorUp:
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
							ct.Print(ansi.CursorMove(n - cursor))
							cursor = n
						}
					}
				case easyterm.CursorDown:
					// move down through command history
					if len(ct.commandHistory) > 0 {
						if history < len(ct.commandHistory)-1 {
							history++
							copy(input, ct.commandHistory[history].input)
							n = len(ct.commandHistory[history].input)
							ct.Print(ansi.CursorMove(n - cursor))
							cursor = n
						} else if history == len(ct.commandHistory)-1 {
							history++
							copy(input, buffInput)
							n = buffN
							ct.Print(ansi.CursorMove(n - cursor))
							cursor = n
						}
					}
				case easyterm.CursorForward:
					// move forward through current command input
					if cursor < n {
						ct.Print(ansi.CursorForwardOne)
						cursor++
					}
				case easyterm.CursorBackward:
					// move backward through current command input
					if cursor > 0 {
						ct.Print(ansi.CursorBackwardOne)
						cursor--
					}

				case easyterm.EscDelete:
					// DELETE
					if cursor < n {
						copy(input[cursor:], input[cursor+1:])
						cursor--
						n--
						history = len(ct.commandHistory)
					}
				}
			}

		case easyterm.KeyBackspace:
			// BACKSPACE
			if cursor > 0 {
				copy(input[cursor-1:], input[cursor:])
				ct.Print(ansi.CursorBackwardOne)
				cursor--
				n--
				history = len(ct.commandHistory)
			}

		default:
			if unicode.IsPrint(r) {
				ct.Print("%c", r)
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
