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
	b.WriteString(fmt.Sprintf("%s->%s", bk.target.ShortLabel(), bk.target.FormatValue(bk.value)))
	n := bk.next
	for n != nil {
		b.WriteString(fmt.Sprintf(" & %s->%s", n.target.ShortLabel(), bk.target.FormatValue(n.value)))
		n = n.next
	}
	return b.String()
}

// isSingleton checks if break condition is part of a list (false) or is a
// singleton condition (true)
func (bk breaker) isSingleton() bool {
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
		if bk.isSingleton() {
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

func (bp *breakpoints) drop(num int) error {
	if len(bp.breaks)-1 < num {
		return errors.NewGopherError(errors.CommandError, fmt.Errorf("breakpoint #%d is not defined", num))
	}

	h := bp.breaks[:num]
	t := bp.breaks[num+1:]
	bp.breaks = make([]breaker, len(h)+len(t), cap(bp.breaks))
	copy(bp.breaks, h)
	copy(bp.breaks[len(h):], t)

	return nil
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
			bp.dbg.print(ui.Feedback, "% 2d: %s", i, bp.breaks[i])
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

	// resolvedTarget keeps track of whether we have specified a target but not
	// given any values for that target. we set it to true initially because
	// we want to be able to change the default target
	resolvedTarget := true

	// we don't add new breakpoints to the main list straight away. we append
	// them to newBreaks first and then check that we aren't adding duplicates
	newBreaks := make([]breaker, 0, 10)

	// loop over tokens. if token is a number then add the breakpoint for the
	// current target. if it is not a number, look for a keyword that changes
	// the target (or run a BREAK meta-command)
	tok, present := tokens.Get()
	for present {
		// if token is a number...
		val, err := strconv.ParseInt(tok, 0, 16)
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
			// if token is not a number ...

			// make sure we've not left a previous target dangling without a value
			if !resolvedTarget {
				return errors.NewGopherError(errors.CommandError, fmt.Errorf("need a value to break on (%s)", tgt.Label()))
			}

			// possibly switch composition mode
			if tok == "&" {
				andBreaks = true
			} else if tok == "|" {
				andBreaks = false
			} else {
				// token is not a number or a composition symbol so try to
				// parse a new target
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
		return errors.NewGopherError(errors.CommandError, fmt.Errorf("need a value to break on (%s)", tgt.Label()))
	}

	// don't add breakpoints that already exist
	duplicate := false
	for _, nb := range newBreaks {
		for _, ob := range bp.breaks {
			and := &nb
			oand := &ob
			for !duplicate && and != nil && oand != nil {
				// note that this method of duplication detection only works if
				// targets are ANDed in the same order.
				// TODO: sort conditions before comparison
				duplicate = oand.target.Label() == and.target.Label() && oand.value == and.value
				and = and.next
				oand = oand.next
			}
			if duplicate {
				break
			}
		}

		// fail on first error
		if duplicate {
			return errors.NewGopherError(errors.CommandError, fmt.Errorf("breakpoint already exists (%s)", nb))
		}

		bp.breaks = append(bp.breaks, nb)
	}

	return nil
}
