package parser

import (
	"strings"
	"time"
)

const cycleDuration = 500 * time.Millisecond

// TabCompletion keeps track of the most recent tab completion attempt
type TabCompletion struct {
	baseOptions Commands

	options    []string
	lastOption int

	// lastGuess is the last string generated and returned by the GuessWord
	// function. we use it to help decide whether to start a new completion
	// session
	lastGuess string

	lastCompletionTime time.Time
}

// NewTabCompletion is the preferred method of initialisation for TabCompletion
func NewTabCompletion(baseOptions Commands) *TabCompletion {
	tc := new(TabCompletion)
	tc.baseOptions = baseOptions
	tc.options = make([]string, 0, len(tc.baseOptions))
	return tc
}

// GuessWord transforms the input such that the last word in the input is
// expanded to meet the closest match in the list of allowed strings.
func (tc *TabCompletion) GuessWord(input string) string {
	// split input into words
	p := strings.Fields(input)
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

		// get args for command
		var arg Arg

		argList, ok := tc.baseOptions[strings.ToUpper(p[0])]
		if ok {
			if len(p)-1 > len(argList) {
				return input
			}

			if len(p) == 1 {
				arg = argList[len(p)-1]
			} else {
				arg = argList[len(p)-2]
			}
		} else {
			arg.Typ = ArgKeyword
			arg.Vals = &tc.baseOptions
		}

		switch arg.Typ {
		case ArgKeyword:
			// trigger is the word we're trying to complete on
			trigger := strings.ToUpper(p[len(p)-1])
			p = p[:len(p)-1]

			switch kw := arg.Vals.(type) {
			case *Commands:
				for k := range *kw {
					if len(trigger) <= len(k) && trigger == k[:len(trigger)] {
						tc.options = append(tc.options, k)
					}
				}
			case Keywords:
				for _, k := range kw {
					if len(trigger) <= len(k) && trigger == k[:len(trigger)] {
						tc.options = append(tc.options, k)
					}
				}
			default:
				tc.options = append(tc.options, "unhandled argument type")
			}

		case ArgFile:
			// TODO: filename completion
			tc.options = append(tc.options, "<TODO: file-completion>")
		}

		// no completion options - return input unchanged
		if len(tc.options) == 0 {
			return input
		}

	}

	// add guessed word to end of input-list and rejoin to form the output
	p = append(p, tc.options[tc.lastOption])
	tc.lastGuess = strings.Join(p, " ") + " "

	// note current time. we'll use this to help decide whether to cycle
	// through a list of options or to begin a new completion session
	tc.lastCompletionTime = time.Now()

	return tc.lastGuess
}
