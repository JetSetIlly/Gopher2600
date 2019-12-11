// breakpoints are used to halt execution when a  target is *changed to* a
// specific value.  compare to traps which are used to halt execution when the
// target *changes from* its current value *to* any other value.

package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/commandline"
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
}

func (bk breaker) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s->%s", bk.target.Label(), bk.target.FormatValue(bk.value)))
	n := bk.next
	for n != nil {
		s.WriteString(fmt.Sprintf(" & %s->%s", n.target.Label(), n.target.FormatValue(n.value)))
		n = n.next
	}
	return s.String()
}

// breaker.check checks the specific break condition with the current value of
// the break target
func (bk *breaker) check() bool {
	currVal := bk.target.CurrentValue()
	m := currVal == bk.value
	if !m {
		bk.ignoreValue = nil
		return false
	}

	if currVal == bk.ignoreValue {
		return false
	}

	if bk.next != nil {
		if !bk.next.check() {
			return false
		}
	}

	bk.ignoreValue = currVal

	return true
}

// breaker.add links a new breaker object to an existing breaker object
func (bk *breaker) add(nbk *breaker) {
	n := &bk.next
	for *n != nil {
		*n = (*n).next
	}
	*n = nbk
}

// newBreakpoints is the preferred method of initialisation for breakpoins
func newBreakpoints(dbg *Debugger) *breakpoints {
	bp := &breakpoints{dbg: dbg}
	bp.clear()
	return bp
}

func (bp *breakpoints) clear() {
	bp.breaks = make([]breaker, 0, 10)
}

func (bp *breakpoints) drop(num int) error {
	if len(bp.breaks)-1 < num {
		return errors.New(errors.CommandError, fmt.Sprintf("breakpoint #%d is not defined", num))
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
		bp.dbg.print(terminal.StyleFeedback, "no breakpoints")
	} else {
		bp.dbg.print(terminal.StyleFeedback, "breakpoints")
		for i := range bp.breaks {
			bp.dbg.print(terminal.StyleFeedback, "% 2d: %s", i, bp.breaks[i])
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
// !!TODO: more sophisticated breakpoints parser
func (bp *breakpoints) parseBreakpoint(tokens *commandline.Tokens) error {
	andBreaks := false

	// default target of CPU PC. meaning that "BREAK n" will cause a breakpoint
	// being set on the PC. breaking on PC is probably the most common type of
	// breakpoint. the target will change value when the input string sees
	// something appropriate
	tgt := target(bp.dbg.vcs.CPU.PC)

	// resolvedTarget keeps track of whether we have specified a target but not
	// given any values for that target. we set it to true initially because
	// we want to be able to change the default target
	resolvedTarget := true

	// we don't add new breakpoints to the main list straight away. we append
	// them to newBreaks first and then check that we aren't adding duplicates
	newBreaks := make([]breaker, 0, 10)

	// loop over tokens:
	//	o if token is a valid type value then add the breakpoint for the current target
	//  o if it is not a valid type value, try to change the target
	tok, present := tokens.Get()
	for present {
		var val interface{}
		var err error

		// try to interpret the token depending on the type of value the target
		// expects
		switch tgt.CurrentValue().(type) {
		case int:
			var v int64
			v, err = strconv.ParseInt(tok, 0, 32)
			if err == nil {
				val = int(v)
			}
		case bool:
			switch strings.ToLower(tok) {
			case "true":
				val = true
			case "false":
				val = false
			default:
				err = errors.New(errors.CommandError, fmt.Sprintf("invalid value (%s) for target (%s)", tok, tgt.Label()))
			}
		default:
			return errors.New(errors.CommandError, fmt.Sprintf("unsupported value type (%T) for target (%s)", tgt.CurrentValue(), tgt.Label()))
		}

		if err == nil {
			if andBreaks {
				if len(newBreaks) == 0 {
					newBreaks = append(newBreaks, breaker{target: tgt, value: val})
				} else {
					newBreaks[len(newBreaks)-1].add(&breaker{target: tgt, value: val})
				}
				resolvedTarget = true
			} else {
				newBreaks = append(newBreaks, breaker{target: tgt, value: val})
				resolvedTarget = true
			}

		} else {
			// make sure we've not left a previous target dangling without a value
			if !resolvedTarget {
				return errors.New(errors.CommandError, err)
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
					return errors.New(errors.CommandError, err)
				}
				resolvedTarget = false
			}
		}

		tok, present = tokens.Get()
	}

	if !resolvedTarget {
		return errors.New(errors.CommandError, fmt.Sprintf("need a value (%T) to break on (%s)", tgt.CurrentValue(), tgt.Label()))
	}

	return bp.checkNewBreakpoints(newBreaks)
}

func (bp *breakpoints) checkNewBreakpoints(newBreaks []breaker) error {
	// don't add breakpoints that already exist
	for _, nb := range newBreaks {
		for _, ob := range bp.breaks {
			and := &nb
			oand := &ob

			// start with assuming this is a duplicate
			duplicate := true

			// continue comparison until we reach the end of one of the lists
			// or if a non-duplicate condition has been found
			for duplicate && and != nil && oand != nil {
				// note that this method of duplication detection only works if
				// targets are ANDed in the same order.
				//
				// !!TODO: sort conditions before comparison
				duplicate = duplicate && (oand.target.Label() == and.target.Label() && oand.value == and.value)

				and = and.next
				oand = oand.next
			}

			// fail if this is a duplicate and if both lists were of the same length
			if duplicate && and == nil && oand == nil {
				return errors.New(errors.CommandError, fmt.Sprintf("breakpoint already exists (%s)", ob))
			}
		}

		bp.breaks = append(bp.breaks, nb)
	}

	return nil
}
