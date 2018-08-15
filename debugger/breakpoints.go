// breakpoints are used to halt execution when a  target is *changed to* a
// specific value.  compare to traps which are used to halt execution when the
// target *changes from* its current value *to* any other value.

package debugger

import (
	"fmt"
	"gopher2600/debugger/input"
	"gopher2600/debugger/ui"
	"gopher2600/errors"
	"strconv"
	"strings"
)

// breakpoints keeps track of all the currently defined breakers
type breakpoints struct {
	dbg    *Debugger
	breaks []breaker
}

// breaker defines a specific break condition
type breaker struct {
	target      target
	value       interface{}
	ignoreValue interface{}

	// basic linked list to implement AND-conditions
	next *breaker
	prev *breaker
}

func (bk breaker) String() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("%s->%d", bk.target.ShortLabel(), bk.value))
	n := bk.next
	for n != nil {
		b.WriteString(fmt.Sprintf(" & %s->%d", n.target.ShortLabel(), n.value))
		n = n.next
	}
	return b.String()
}

// isSingleton checks if break condition is part of a list (false) or is a
// singleton condition (true)
func (bk breaker) isSinglton() bool {
	return bk.next == nil && bk.prev == nil
}

// breaker.check checks the specific break condition with the current value of
// the break target
func (bk *breaker) check() bool {
	currVal := bk.target.Value()
	b := currVal == bk.value

	if bk.next == nil {
		b = b && currVal != bk.ignoreValue

		// this is either a singleton break or the end of a break-list
		// (inList==true). note how we set the ignoreValue in these two
		// instances. if it's a singleton break then we always reset the
		// ignoreValue. if it's the end of the list we reset the value to nil
		// if there is no match
		if bk.isSinglton() {
			bk.ignoreValue = currVal
		} else {
			bk.ignoreValue = nil
		}
		return b
	}

	// this breaker is part of list so we need to recurse into the list
	b = b && bk.next.check()
	if b {
		b = b && currVal != bk.ignoreValue
		bk.ignoreValue = currVal
	} else {
		bk.ignoreValue = nil
	}

	return b
}

// add appends a new breaker object to the *end of the list* from the perspective
// of bk
func (bk *breaker) add(nbk *breaker) {
	n := &bk.next
	for *n != nil {
		nbk.prev = *n
		*n = (*n).next
	}
	*n = nbk
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

// breakpoints.check compares the current state of the emulation with every
// break condition. it returns a string listing every condition that applies
func (bp *breakpoints) check(previousResult string) string {
	checkString := strings.Builder{}
	checkString.WriteString(previousResult)
	for i := range bp.breaks {
		// check current value of target with the requested value
		if bp.breaks[i].check() {
			checkString.WriteString(fmt.Sprintf("break on %s\n", bp.breaks[i]))
		}
	}
	return checkString.String()
}

func (bp breakpoints) list() {
	if len(bp.breaks) == 0 {
		bp.dbg.print(ui.Feedback, "no breakpoints")
	} else {
		for i := range bp.breaks {
			bp.dbg.print(ui.Feedback, "%s", bp.breaks[i])
		}
	}
}

// parseBreakpoints consumes tokens and adds new conditions to the list of
// breakpoints. For example:
//
//	PC 0xf000
//  adds a new breakpoint to the PC
//
//  0xf000
//  is the same, because we assume a target of PC if none is given
//
//  X 10 11
//  adds two new breakpoints to X - we've changed targets so the second value
//  is assumed to be for the previously selected target
//
//  X 10 11 Y 12
//  add three breakpoints; 2 to X and 1 to Y
//
//  SL 100 & HP 0
//  add one AND-condition
//
//  SL 100 & HP 0 | X 10
//  add two conditions; one AND-condition and one condition on X
//
// note that this is a very simple parser and we can do unusual things: the &
// and | symbols simply switch "modes", with unusual consequences. for example,
// the last example above could be written:
//
//  & SL 100 HP 0 | X 10
//
// TODO: more sophisticated breakpoints parser
func (bp *breakpoints) parseBreakpoint(tokens *input.Tokens) error {
	andBreaks := false

	// default target of CPU PC. meaning that "BREAK n" will cause a breakpoint
	// being set on the PC. breaking on PC is probably the most common type of
	// breakpoint. the target will change value when the input string sees
	// something appropriate
	tgt := target(bp.dbg.vcs.MC.PC)

	// resolvedTarget is true to begin with so that the initial target of PC
	// can be changed immediately
	resolvedTarget := true

	// we don't add new breakpoints to the main list straight away. we append
	// them to newBreaks first and then check that we aren't adding duplicates
	newBreaks := make([]breaker, 0, 10)

	// loop over tokens. if token is a number then add the breakpoint for the
	// current target. if it is not a number, look for a keyword that changes
	// the target (or run a BREAK meta-command)
	//
	// note that this method of looping allows the user to chain break commands
	tok, present := tokens.Get()
	for present {
		val, err := strconv.ParseUint(tok, 0, 16)
		if err == nil {
			if andBreaks == true {
				if len(newBreaks) == 0 {
					newBreaks = append(newBreaks, breaker{target: tgt, value: int(val)})
				} else {
					newBreaks[len(newBreaks)-1].add(&breaker{target: tgt, value: int(val)})
				}
				resolvedTarget = true
			} else {
				newBreaks = append(newBreaks, breaker{target: tgt, value: int(val)})
				resolvedTarget = true
			}

		} else {
			if !resolvedTarget {
				return errors.NewGopherError(errors.InputTooFewArgs, fmt.Errorf("need a value to break on (%s)", tgt.Label()))
			}

			if tok == "&" {
				andBreaks = true
			} else if tok == "|" {
				andBreaks = false
			} else {
				tokens.Unget()
				tgt, err = parseTarget(bp.dbg, tokens)
				if err != nil {
					return err
				}
				resolvedTarget = false
			}
		}

		tok, present = tokens.Get()
	}

	if !resolvedTarget {
		return errors.NewGopherError(errors.InputTooFewArgs, fmt.Errorf("need a value to break on (%s)", tgt.Label()))
	}

	// don't add breakpoints that already exist (only works correctly with
	// singleton breaks currently)
	// TODO: fix this so we do not add AND-conditions that already exist
	for _, nb := range newBreaks {
		if nb.next == nil {
			exists := false
			for _, ob := range bp.breaks {
				if ob.next == nil && ob.target == nb.target && ob.value == nb.value {
					bp.dbg.print(ui.Feedback, "breakpoint already exists (%s)", ob)
					exists = true
				}
			}
			if !exists {
				bp.breaks = append(bp.breaks, nb)
			}
		} else {
			bp.breaks = append(bp.breaks, nb)
		}
	}

	return nil
}
