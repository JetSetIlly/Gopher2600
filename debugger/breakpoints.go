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
	dbg *Debugger

	// array of breakers are ORed together
	breaks []breaker
}

// breaker defines a specific break condition
type breaker struct {
	target      target
	value       interface{}
	ignoreValue interface{}

	// single linked list ANDs breakers together
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

// id creates a sum of the breaker sequence such that the order of the sequence
// does not matter. this commutative property makes it useful to detect
// duplicate sequences of ANDed breakers.
//
// note that id collisions using this method is likely if we were applying it
// to arbitrary strings. but given the restrictions on what is a breakpoint
// string and the packing of string length into the LSB, the chances are
// reduced. still, it's something we should be mindful of.
func (bk breaker) id() int {
	// summation of data in each node
	sum := 0

	// number of nodes encountered
	c := 1

	// visit every node in the sequence
	n := &bk
	for n != nil {

		// add the ASCII value of each character in the target label to the sum
		s := n.target.Label()
		for i := 0; i < len(s); i++ {
			sum += int(s[i])
		}

		// add the breakpoint value to the sum
		switch v := n.value.(type) {
		case int:
			sum += v
		case bool:
			// if value type is boolean add one if value is true
			if v {
				sum++
			}
		default:
		}

		n = n.next
		c++
	}

	// stuff number of nodes into the LSB
	return (sum << 8) | (c % 256)
}

// check checks the specific break condition with the current value of
// the break target
func (bk *breaker) check() bool {
	currVal := bk.target.TargetValue()
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

// add a new breaker by linking it to the end of an existing breaker
func (bk *breaker) add(nbk *breaker) {
	n := bk
	for n.next != nil {
		n = n.next
	}
	n.next = nbk
}

// newBreakpoints is the preferred method of initialisation for breakpoints
func newBreakpoints(dbg *Debugger) *breakpoints {
	bp := &breakpoints{dbg: dbg}
	bp.clear()
	return bp
}

// clear all breakpoints
func (bp *breakpoints) clear() {
	bp.breaks = make([]breaker, 0, 10)
}

// drop a specific breakpoint by position in list
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

// check compares the current state of the emulation with every breakpoint
// condition. returns a string listing every condition that matches (separated
// by \n)
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

// list currently defined breakpoints
func (bp breakpoints) list() {
	if len(bp.breaks) == 0 {
		bp.dbg.printLine(terminal.StyleFeedback, "no breakpoints")
	} else {
		bp.dbg.printLine(terminal.StyleFeedback, "breakpoints:")
		for i := range bp.breaks {
			bp.dbg.printLine(terminal.StyleFeedback, "% 2d: %s", i, bp.breaks[i])
		}
	}
}

// parse token and add new breakpoint. for example:
//
//	PC 0xf000
//  adds a new breakpoint to the PC
//
// in addition to the description in the HELP file, the breakpoint parser has
// some additional features which should probably be removed. if only because
// the commandline template will balk before this function is ever called.
//
// for reference though, and very briefly: the | symbol can be used to add more
// than one condition, instead of calling BREAK more than once.
//
// Also, the & symbol can be placed before the target/value combinations.
// A sort of Polish prefix notation.
//
//	& SL 100 HP 0 X 10
//
// !!TODO: simplify breakpoints parser to match help description
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
		switch tgt.TargetValue().(type) {
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
			return errors.New(errors.CommandError, fmt.Sprintf("unsupported value type (%T) for target (%s)", tgt.TargetValue(), tgt.Label()))
		}

		if err == nil {
			if andBreaks {
				newBreaks[len(newBreaks)-1].add(&breaker{target: tgt, value: val})
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
		return errors.New(errors.CommandError, fmt.Sprintf("need a value (%T) to break on (%s)", tgt.TargetValue(), tgt.Label()))
	}

	return bp.checkNewBreakers(newBreaks)
}

// checkNewBreakers compares list of new breakers with existing list
func (bp *breakpoints) checkNewBreakers(newBreaks []breaker) error {
	// don't add breakpoints that already exist
	for _, nb := range newBreaks {
		for _, ob := range bp.breaks {
			if nb.id() == ob.id() {
				return errors.New(errors.CommandError, fmt.Sprintf("breakpoint already exists (%s)", ob))
			}
		}

		bp.breaks = append(bp.breaks, nb)
	}

	return nil
}
