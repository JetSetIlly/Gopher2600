// traps are used to halt execution of the emulator when the target *changes*
// from its current value to any other value. compare to breakpoints which halt
// execution when the target is *changed to* a specific value.

package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/commandline"
	"gopher2600/errors"
	"strings"
)

type traps struct {
	dbg   *Debugger
	traps []trapper
}

type trapper struct {
	target    target
	origValue interface{}
}

func (tr trapper) String() string {
	return tr.target.Label()
}

// newTraps is the preferred method of initialisation for the traps type
func newTraps(dbg *Debugger) *traps {
	tr := &traps{dbg: dbg}
	tr.clear()
	return tr
}

// clear all traps
func (tr *traps) clear() {
	tr.traps = make([]trapper, 0, 10)
}

// drop the numbered trap from the list
func (tr *traps) drop(num int) error {
	if len(tr.traps)-1 < num {
		return errors.New(errors.CommandError, fmt.Sprintf("trap #%d is not defined", num))
	}

	h := tr.traps[:num]
	t := tr.traps[num+1:]
	tr.traps = make([]trapper, len(h)+len(t), cap(tr.traps))
	copy(tr.traps, h)
	copy(tr.traps[len(h):], t)

	return nil
}

// check compares the current state of the emulation with every trap condition.
// returns a string listing every condition that matches (separated by \n)
func (tr *traps) check(previousResult string) string {
	checkString := strings.Builder{}
	checkString.WriteString(previousResult)
	for i := range tr.traps {
		trapValue := tr.traps[i].target.TargetValue()

		if trapValue != tr.traps[i].origValue {
			checkString.WriteString(fmt.Sprintf("trap on %s [%v->%v]\n", tr.traps[i].target.Label(), tr.traps[i].origValue, trapValue))
			tr.traps[i].origValue = trapValue
		}
	}
	return checkString.String()
}

// list currently defined traps
func (tr traps) list() {
	if len(tr.traps) == 0 {
		tr.dbg.print(terminal.StyleFeedback, "no traps")
	} else {
		tr.dbg.print(terminal.StyleFeedback, "traps")
		for i := range tr.traps {
			tr.dbg.print(terminal.StyleFeedback, "% 2d: %s", i, tr.traps[i].target.Label())
		}
	}
}

// parse tokens and add new trap
func (tr *traps) parseTrap(tokens *commandline.Tokens) error {
	_, present := tokens.Peek()
	for present {
		tgt, err := parseTarget(tr.dbg, tokens)
		if err != nil {
			return err
		}

		addNewTrap := true
		for _, t := range tr.traps {
			if t.target.Label() == tgt.Label() {
				addNewTrap = false
				tr.dbg.print(terminal.StyleError, fmt.Sprintf("trap already exists (%s)", t))
				break // for loop
			}
		}

		if addNewTrap {
			tr.traps = append(tr.traps, trapper{target: tgt, origValue: tgt.TargetValue()})
		}

		_, present = tokens.Peek()
	}

	return nil
}
