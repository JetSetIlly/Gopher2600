package debugger

import (
	"fmt"
	"gopher2600/debugger/ui"
	"strconv"
)

// breakpoints keeps track of all the currently defined breaker
type breakpoints struct {
	dbg    *Debugger
	breaks []breaker

	// ignore certain target values
	ignoredBreakerStates map[target]int
}

// breaker defines a specific break condition
type breaker struct {
	target target
	value  int
}

// newBreakpoints is the preferred method of initialisation for breakpoins
func newBreakpoints(dbg *Debugger) *breakpoints {
	bp := new(breakpoints)
	bp.dbg = dbg
	bp.clear()
	return bp
}

func (bp *breakpoints) clear() {
	bp.breaks = make([]breaker, 0, 10)
}

// prepareBreakpoints prepares for next breakpoint by storing the current state
// of all Targets. we can then use these stored values to know what to
// ignore. used primarily so that we're not breaking immediately on a previous
// breakstate.
//
// one possible flaw in the current implementation of this idea is that the
// emulation will not honour new breaks until the value has cycled back to the
// break value:
//
//    A == v
//		break A v
//    A == v -> no break
//    A == w -> no break
//		A == v -> breaks
//
func (bp *breakpoints) prepareBreakpoints() {
	bp.ignoredBreakerStates = make(map[target]int, len(bp.breaks))
	for _, b := range bp.breaks {
		bp.ignoredBreakerStates[b.target] = b.target.ToInt()
	}
}

// check compares the current state of the emulation with every break
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (bp *breakpoints) check() bool {
	broken := false
	for i := range bp.breaks {
		if bp.breaks[i].target.ToInt() == bp.breaks[i].value {
			// make sure that we're not breaking on an ignore state
			bv, prs := bp.ignoredBreakerStates[bp.breaks[i].target]
			if !prs || prs && bp.breaks[i].target.ToInt() != bv {
				bp.dbg.print(ui.Feedback, "break on %s=%d", bp.breaks[i].target.ShortLabel(), bp.breaks[i].value)
				broken = true
			}
		}
	}

	// remove ignoreBreakerState if the break target has changed from its
	// ignored value
	if !broken {
		for i := range bp.breaks {
			bv, prs := bp.ignoredBreakerStates[bp.breaks[i].target]
			if prs && bp.breaks[i].target.ToInt() != bv {
				delete(bp.ignoredBreakerStates, bp.breaks[i].target)
			}
		}
	}

	return broken
}

func (bp breakpoints) list() {
	if len(bp.breaks) == 0 {
		bp.dbg.print(ui.Feedback, "no breakpoints")
	} else {
		for i := range bp.breaks {
			bp.dbg.print(ui.Feedback, "%s->%d", bp.breaks[i].target.ShortLabel(), bp.breaks[i].value)
		}
	}
}

func (bp *breakpoints) parseBreakpoint(parts []string) error {
	if len(parts) == 1 {
		return fmt.Errorf("not enough arguments for %s", parts[0])
	}

	var tgt target

	// default target of CPU PC. meaning that "BREAK n" will cause a breakpoint
	// being set on the PC. breaking on PC is probably the most common type of
	// breakpoint. the target will change value when the input string sees
	// something appropriate
	tgt = bp.dbg.vcs.MC.PC

	// loop over parts. if part is a number then add the breakpoint for the
	// current target. if it is not a number, look for a keyword that changes
	// the target (or run a BREAK meta-command)
	//
	// note that this method of looping allows the user to chain break commands
	for i := 1; i < len(parts); i++ {

		val, err := strconv.ParseUint(parts[i], 0, 16)
		if err == nil {
			// check to see if breakpoint already exists
			addNewBreak := true
			for _, mv := range bp.breaks {
				if mv.target == tgt && mv.value == int(val) {
					addNewBreak = false
					bp.dbg.print(ui.Feedback, "breakpoint already exists")
					break // for loop
				}
			}
			if addNewBreak {
				bp.breaks = append(bp.breaks, breaker{target: tgt, value: int(val)})
			}

		} else {

			// TODO: namespaces so we can do things like "BREAK TV COLOR RED" without
			// our breakpoints code knowing anything about it. GetTVState() will
			// return a TVState if the television implementation understands the
			// request

			// commands
			switch parts[i] {
			case "CLEAR":
				bp.clear()
				bp.dbg.print(ui.Feedback, "breakpoints cleared")
				return nil
			case "LIST":
				bp.list()
				return nil
			}

			// defer parsing of other keywords to parseTargets()
			tgt = parseTarget(bp.dbg.vcs, parts[i])
			if tgt == nil {
				return fmt.Errorf("invalid %s target (%s)", parts[0], parts[i])
			}
		}
	}

	return nil
}
