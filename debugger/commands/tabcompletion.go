package commands

import (
	"strings"
	"time"
)

const cycleDuration = 500 * time.Millisecond

// TabCompletion keeps track of the most recent tab completion attempt
type TabCompletion struct {
	options    []string
	lastOption int

	// lastGuess is the last string generated and returned by the GuessWord
	// function. we use it to help decide whether to start a new completion
	// session
	lastGuess string

	lastCompletionTime time.Time
}

// NewTabCompletion is the preferred method of initialisation for TabCompletion
func NewTabCompletion() *TabCompletion {
	tc := new(TabCompletion)
	tc.options = make([]string, 0, len(DebuggerCommand))
	return tc
}

// GuessWord transforms the input such that the last word in the input is
// expanded to meet the closest match in the list of allowed strings.
func (tc *TabCompletion) GuessWord(input string) string {
	// split input into words
	p := strings.Split(input, " ")
	if len(p) == 0 {
		return input
	}

	// if input string is the same as the string last returned by this function
	// AND it is within a time duration of 'cycleDuration' then return the next
	// option
	if input == tc.lastGuess && time.Since(tc.lastCompletionTime) < cycleDuration {

		// if there was only one option in the option list then return immediatly
		if len(tc.options) <= 1 {
			return input
		}

		// there is more than one completion option, so shorten the input by
		// one word (getting rid of the last completion effort) and step to
		// next option
		p = p[:len(p)-1]
		tc.lastOption++
		if tc.lastOption >= len(tc.options) {
			tc.lastOption = 0
		}

	} else {
		// this is a new tabcompletion session
		tc.options = tc.options[:0]
		tc.lastOption = 0

		context := completionsOpts[strings.ToUpper(p[0])]

		if len(p) == 0 || context == compArgDebuggerCommand {
			trigger := strings.ToUpper(p[len(p)-1])
			// if this is the first word in the input or if the completion is
			// otherwise suitable, build a list of options formed from the list of
			// debugger commands
			for i := 0; i < len(DebuggerCommand); i++ {
				if len(trigger) <= len(DebuggerCommand[i]) && trigger == DebuggerCommand[i][:len(trigger)] {
					tc.options = append(tc.options, DebuggerCommand[i])
				}
			}
		} else if context == compArgFile {
			// TODO: filename completion
			tc.options = append(tc.options, "<TODO: file-completion>")
		}

		// no completion options - return input unchanged
		if len(tc.options) == 0 {
			return input
		}
	}

	// change the last word in the supplied input to the chosen option
	p[len(p)-1] = tc.options[tc.lastOption]

	// rejoin all parts of the input along with the altered last word
	tc.lastGuess = strings.Join(p, " ") + " "

	// note current time. we'll use this to help decide whether to cycle
	// through a list of options or to begin a new completion session
	tc.lastCompletionTime = time.Now()

	return tc.lastGuess
}
