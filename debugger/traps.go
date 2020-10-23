// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

// traps are used to halt execution of the emulator when the target *changes*
// from its current value to any other value. compare to breakpoints which halt
// execution when the target is *changed to* a specific value.

package debugger

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
)

type traps struct {
	dbg   *Debugger
	traps []trapper
}

type trapper struct {
	target    *target
	origValue interface{}
}

func (tr trapper) String() string {
	return tr.target.Label()
}

// newTraps is the preferred method of initialisation for the traps type.
func newTraps(dbg *Debugger) *traps {
	tr := &traps{dbg: dbg}
	tr.clear()
	return tr
}

// clear all traps.
func (tr *traps) clear() {
	tr.traps = make([]trapper, 0, 10)
}

// isEmpty returns true if there are no currently defined traps.
func (tr *traps) isEmpty() bool {
	return len(tr.traps) == 0
}

// drop the numbered trap from the list.
func (tr *traps) drop(num int) error {
	if len(tr.traps)-1 < num {
		return curated.Errorf("trap #%d is not defined", num)
	}

	h := tr.traps[:num]
	t := tr.traps[num+1:]
	tr.traps = make([]trapper, len(h)+len(t), cap(tr.traps))
	copy(tr.traps, h)
	copy(tr.traps[len(h):], t)

	return nil
}

// check compares the current state of the emulation with every trap condition.
// returns a string listing every condition that matches (separated by \n).
func (tr *traps) check(previousResult string) string {
	if len(tr.traps) == 0 {
		return previousResult
	}

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

// list currently defined traps.
func (tr traps) list() {
	if len(tr.traps) == 0 {
		tr.dbg.printLine(terminal.StyleFeedback, "no traps")
	} else {
		tr.dbg.printLine(terminal.StyleFeedback, "traps:")
		for i := range tr.traps {
			tr.dbg.printLine(terminal.StyleFeedback, "% 2d: %s", i, tr.traps[i].target.Label())
		}
	}
}

// parse tokens and add new trap.
func (tr *traps) parseCommand(tokens *commandline.Tokens) error {
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
				tr.dbg.printLine(terminal.StyleError, fmt.Sprintf("trap already exists (%s)", t))
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
