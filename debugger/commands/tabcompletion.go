package commands

import (
	"strings"
	"time"
)

const cycleDuration = 1500 * time.Millisecond

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
	tc.options = make([]string, 0, len(TopLevel))
	return tc
}

// GuessWord transforms the input such that the word nearest the cursor is
// expanded to meet the closest match in the list of allowed strings
// returns: the input with the completed word; the number of characters by
// which the cursor should be transformed (may be a negative number when
// cycling through a list of options)
// TODO: filename completion for commands that need it (eg. script)
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

		// undo previous tab completion
		p = p[:len(p)-1]

		// step to next option; we'll build the complete return string below
		tc.lastOption++
		if tc.lastOption >= len(tc.options) {
			tc.lastOption = 0
		}

	} else {
		// this is a new tabcompletion session
		trigger := strings.ToUpper(p[len(p)-1])
		tc.options = tc.options[:0]
		tc.lastOption = 0

		// build a list of options
		for i := 0; i < len(TopLevel); i++ {
			if len(trigger) <= len(TopLevel[i]) && trigger == TopLevel[i][:len(trigger)] {
				tc.options = append(tc.options, TopLevel[i])
			}
		}
	}

	// no completion options - return input unchanged
	if len(tc.options) == 0 {
		return input
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
