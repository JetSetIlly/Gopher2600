// traps are used to halt execution of the emulator when the target *changes*
// from its current value to any other value. compare to breakpoints which halt
// execution when the target is *changed to* a specific value.

package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"strings"
)

// traps keeps track of all the currently defined trappers
type traps struct {
	dbg   *Debugger
	traps []trapper
}

// trapper defines a specific trap
type trapper struct {
	target    target
	origValue interface{}
}

// newTraps is the preferred method of initialisation for traps
func newTraps(dbg *Debugger) *traps {
	tr := new(traps)
	tr.dbg = dbg
	tr.clear()
	return tr
}

func (tr *traps) clear() {
	tr.traps = make([]trapper, 0, 10)
}

// check compares the current state of the emulation with every trap condition.
// it returns a string listing every condition that applies
func (tr *traps) check(previousResult string) string {
	checkString := strings.Builder{}
	checkString.WriteString(previousResult)
	for i := range tr.traps {
		if tr.traps[i].target.Value() != tr.traps[i].origValue {
			tr.traps[i].origValue = tr.traps[i].target.Value()
			checkString.WriteString(fmt.Sprintf("trap on %s", tr.traps[i].target.ShortLabel()))
		}
	}
	return checkString.String()
}

func (tr traps) list() {
	if len(tr.traps) == 0 {
		tr.dbg.print(ui.Feedback, "no traps")
	} else {
		s := ""
		sep := ""
		for i := range tr.traps {
			s = fmt.Sprintf("%s%s%s", s, sep, tr.traps[i].target.ShortLabel())
			sep = ", "
		}
		tr.dbg.print(ui.Feedback, s)
	}
}

func (tr *traps) parseTrap(tokens *input.Tokens) error {
	_, present := tokens.Peek()
	for present {
		tgt, err := parseTarget(tr.dbg, tokens)
		if err != nil {
			return err
		}

		addNewTrap := true
		for _, t := range tr.traps {
			if t.target == tgt {
				addNewTrap = false
				tr.dbg.print(ui.Feedback, "trap already exists")
				break // for loop
			}
		}

		if addNewTrap {
			tr.traps = append(tr.traps, trapper{target: tgt, origValue: tgt.Value()})
		}

		_, present = tokens.Peek()
	}

	return nil
}
