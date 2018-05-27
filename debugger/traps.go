package debugger

import (
	"fmt"
)

// breakpoints keeps track of all the currently defined breakers and any
// other special conditions that may interrupt execution
type traps struct {
	dbg   *Debugger
	traps []trapper
}

// breaker defines a specific break condition
type trapper struct {
	target    target
	origValue int
}

// newBreakpoints is the preferred method of initialisation for breakpoins
func newTraps(dbg *Debugger) *traps {
	tr := new(traps)
	tr.dbg = dbg
	tr.clear()
	return tr
}

func (tr *traps) clear() {
	tr.traps = make([]trapper, 0, 10)
}

// check compares the current state of the emulation with every break
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (tr *traps) check() bool {
	trapped := false
	for i := range tr.traps {
		trapped = tr.traps[i].target.ToInt() != tr.traps[i].origValue
		if trapped {
			tr.traps[i].origValue = tr.traps[i].target.ToInt()
			tr.dbg.print(Feedback, "trap on %s", tr.traps[i].target.ShortLabel())
		}
	}

	return trapped
}

func (tr traps) list() {
	if len(tr.traps) == 0 {
		tr.dbg.print(Feedback, "no traps")
	} else {
		s := ""
		sep := ""
		for i := range tr.traps {
			s = fmt.Sprintf("%s%s%s", s, sep, tr.traps[i].target.ShortLabel())
			sep = ", "
		}
		tr.dbg.print(Feedback, s)
	}
}

func (tr *traps) parseTrap(parts []string) error {
	if len(parts) == 1 {
		tr.list()
	}

	// loop over parts, allowing multiple traps to be applied
	for i := 1; i < len(parts); i++ {
		// commands
		switch parts[i] {
		case "CLEAR":
			tr.clear()
			tr.dbg.print(Feedback, "traps cleared")
			return nil
		case "LIST":
			tr.list()
			return nil
		}

		target := parseTarget(tr.dbg.vcs, parts[i])
		tr.traps = append(tr.traps, trapper{target: target, origValue: target.ToInt()})
	}

	return nil
}
