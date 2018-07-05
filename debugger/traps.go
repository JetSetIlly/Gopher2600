package debugger

import (
	"fmt"
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
	origValue int
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

// check compares the current state of the emulation with every trap
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (tr *traps) check() bool {
	trapped := false
	for i := range tr.traps {
		ntr := tr.traps[i].target.ToInt() != tr.traps[i].origValue
		if ntr {
			tr.traps[i].origValue = tr.traps[i].target.ToInt()
			tr.dbg.print(ui.Feedback, "trap on %s", tr.traps[i].target.ShortLabel())
		}
		trapped = ntr || trapped
	}

	return trapped
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

func (tr *traps) parseTrap(parts []string) error {
	// loop over parts, allowing multiple traps to be applied
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i])

		tgt := parseTarget(tr.dbg.vcs, parts[i])
		if tgt == nil {
			return fmt.Errorf("invalid %s target (%s)", parts[0], parts[i])
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
			tr.traps = append(tr.traps, trapper{target: tgt, origValue: tgt.ToInt()})
		}
	}

	return nil
}
