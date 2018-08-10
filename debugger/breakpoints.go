// breakpoints are used to halt execution when a  target is *changed to* a
// specific value.  compare to traps which are used to halt execution when the
// target *changes* from its current value to any other value.

package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/errors"
	"strconv"
)

// breakpoints keeps track of all the currently defined breaker
type breakpoints struct {
	dbg    *Debugger
	breaks []breaker

	// ignore certain target values
	ignoredBreakerStates map[target]interface{}
}

// breaker defines a specific break condition
type breaker struct {
	target target
	value  interface{}
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
func (bp *breakpoints) prepareBreakpoints() {
	bp.ignoredBreakerStates = make(map[target]interface{}, len(bp.breaks))
	for _, b := range bp.breaks {
		bp.ignoredBreakerStates[b.target] = b.target.Value()
	}
}

// check compares the current state of the emulation with every break
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (bp *breakpoints) check() bool {
	broken := false
	for i := range bp.breaks {
		// check current value of target with the requested value
		if bp.breaks[i].target.Value() == bp.breaks[i].value {
			// make sure that we're not breaking on an ignore state
			bv, prs := bp.ignoredBreakerStates[bp.breaks[i].target]
			if !prs || prs && bp.breaks[i].target.Value() != bv {
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
			if prs && bp.breaks[i].target.Value() != bv {
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

func (bp *breakpoints) parseBreakpoint(tokens *input.Tokens) error {
	var tgt target

	// resolvedTarget notes whether a target has been used correctly
	var resolvedTarget bool

	// default target of CPU PC. meaning that "BREAK n" will cause a breakpoint
	// being set on the PC. breaking on PC is probably the most common type of
	// breakpoint. the target will change value when the input string sees
	// something appropriate
	tgt = bp.dbg.vcs.MC.PC

	// resolvedTarget is true to begin with so that the initial target of PC
	// can be changed immediately
	resolvedTarget = true

	// loop over tokens. if token is a number then add the breakpoint for the
	// current target. if it is not a number, look for a keyword that changes
	// the target (or run a BREAK meta-command)
	//
	// note that this method of looping allows the user to chain break commands
	a, present := tokens.Get()
	for present {
		val, err := strconv.ParseUint(a, 0, 16)
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
			resolvedTarget = true
		} else {
			if !resolvedTarget {
				return errors.NewGopherError(errors.InputTooFewArgs, fmt.Errorf("need a value to break on (%s)", tgt.Label()))
			}

			tokens.Unget()
			tgt, err = parseTarget(bp.dbg, tokens)
			if err != nil {
				return err
			}
			resolvedTarget = false
		}

		a, present = tokens.Get()
	}

	if !resolvedTarget {
		return errors.NewGopherError(errors.InputTooFewArgs, fmt.Errorf("need a value to break on (%s)", tgt.Label()))
	}

	return nil
}
