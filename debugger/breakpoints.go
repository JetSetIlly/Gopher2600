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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// breakpoints are used to halt execution when a  target is *changed to* a
// specific value.  compare to traps which are used to halt execution when the
// target *changes from* its current value *to* any other value.

package debugger

import (
	"fmt"
	"gopher2600/debugger/terminal"
	"gopher2600/debugger/terminal/commandline"
	"gopher2600/disassembly"
	"gopher2600/errors"
	"strconv"
	"strings"
)

// breakpoints keeps track of all the currently defined breakers
type breakpoints struct {
	dbg *Debugger

	// array of breakers are ORed together
	breaks []breaker

	// prepared targets which we use to check for PC breaks. see hasPcBreak()
	checkPcBreak       *target
	checkBankBreak     *target
	checkMnemonicBreak *target
}

// breaker defines a specific break condition
type breaker struct {
	target      *target
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

// compares two breakers for equality. returns true if the two breakers are
// logically the same.
func (bk breaker) cmp(ck breaker) bool {
	// count number of nodes
	bn := 0
	b := &bk
	for b != nil {
		bn++
		b = b.next
	}

	cn := 0
	c := &ck
	for c != nil {
		cn++
		c = c.next
	}

	// if counts are different then the comparison has failed
	if cn != bn {
		return false
	}

	// compare all nodes with one another
	b = &bk
	for b != nil {
		c = &ck
		match := false
		for c != nil {
			match = (b.target.label == c.target.label && b.value == c.value)
			if match {
				break // for loop
			}
			c = c.next
		}

		if !match {
			return false
		}

		b = b.next
	}

	return true
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
func newBreakpoints(dbg *Debugger) (*breakpoints, error) {
	bp := &breakpoints{dbg: dbg}
	bp.clear()

	var err error

	bp.checkPcBreak, err = parseTarget(bp.dbg, commandline.TokeniseInput("PC"))
	if err != nil {
		return nil, errors.New(errors.BreakpointError, "fatality while setting up breakpoint parser")
	}

	bp.checkBankBreak, err = parseTarget(bp.dbg, commandline.TokeniseInput("BANK"))
	if err != nil {
		return nil, errors.New(errors.BreakpointError, "fatality while setting up breakpoint parser")
	}

	bp.checkMnemonicBreak, err = parseTarget(bp.dbg, commandline.TokeniseInput("RESULT MNEMONIC"))
	if err != nil {
		return nil, errors.New(errors.BreakpointError, "fatality while setting up breakpoint parser")
	}

	return bp, err
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
	tgt, err := parseTarget(bp.dbg, commandline.TokeniseInput("PC"))
	if err != nil {
		return errors.New(errors.BreakpointError, "fatality while setting up breakpoint parser")
	}

	// resolvedTarget keeps track of whether we have specified a target but not
	// given any values for that target. we set it to true initially because
	// we want to be able to change the default target
	resolvedTarget := true

	// we don't add new breakpoints to the main list straight away. we append
	// them to newBreaks first and then check that we aren't adding duplicates
	newBreaks := make([]breaker, 0, 10)

	// a note about whether the PC target has been specified explicitely. we
	// use this to decide whether to add an automatic BANK condition to PC
	// targets (see below)
	explicitPCTarget := false

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
		case string:
			// if token is string type then make it uppercase for now
			//
			// !!TODO: more sophisticated transforms of breakpoint information
			// see also "special handling for PC" below
			val = strings.ToUpper(tok)
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
			// special handling for PC
			if tgt.Label() == "PC" {
				ai := bp.dbg.dbgmem.mapAddress(uint16(val.(int)), true)
				val = int(ai.mappedAddress)
			}

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
				// note whether PC target has been specified explicitly
				explicitPCTarget = explicitPCTarget || strings.ToUpper(tok) == "PC"

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

	for _, nb := range newBreaks {
		// if the break is a singular, undecorated PC target then add a BANK
		// condition for the current BANK. this is arguably what the user
		// intends to happen.
		if nb.next == nil && nb.target.label == "PC" && !explicitPCTarget {
			if bp.dbg.vcs.Mem.Cart.NumBanks() > 1 {
				nb.next = &breaker{
					target: bankTarget(bp.dbg),
					value:  bp.dbg.vcs.Mem.Cart.GetBank(bp.dbg.vcs.CPU.PC.Address()),
				}
				nb.next.ignoreValue = nb.next.value
			}
		}

		if err := bp.checkBreaker(nb); err != nil {
			return errors.New(errors.CommandError, err)
		}
		bp.breaks = append(bp.breaks, nb)
	}

	return nil
}

func (bp *breakpoints) checkBreaker(nb breaker) error {
	for _, ob := range bp.breaks {
		if nb.cmp(ob) {
			return errors.New(errors.BreakpointError, fmt.Sprintf("already exists (%s)", ob))
		}
	}

	return nil
}

// BreakGroup indicates the broad category of breakpoint an address has
type BreakGroup int

// List of valid BreakGroup values
const (
	BrkGrpNone BreakGroup = iota
	BrkGrpAnyBank
	BrkGrpThisBank
)

// !!TODO: detect other break types?
func (bp *breakpoints) hasBreak(e *disassembly.Entry) BreakGroup {
	ai := bp.dbg.dbgmem.mapAddress(e.Result.Address, true)

	check := breaker{
		target: bp.checkPcBreak,

		// casting value to type because that's how the target value is stored
		// for the program counter (see TargetValue() implementation for the
		// ProgramCounter type in the registers package)
		value: int(ai.mappedAddress),
	}

	// check has slightly different semantics if number of cartridge banks is
	// greater than one. in this case we want to check if PC break specifies a
	// bank or not
	if bp.dbg.vcs.Mem.Cart.NumBanks() > 1 {
		// we start with the very specific - address and bank
		check.next = &breaker{
			target: bp.checkBankBreak,
			value:  e.Bank,
		}

		// check for a breaker for the PC value AND bank value. if
		// checkBreaker() fails then from our point of view, this is a success
		// and we say that the disassembly.Entry has a breakpoint for *this*
		// bank
		if err := bp.checkBreaker(check); err != nil {
			return BrkGrpThisBank
		}

		// if checkBreaker doesn't report an existing breakpoint, we remove the
		// Bank condition and try again. if checkBreaker fails (success from our
		// point of view) this time, we can say that the disassembly entry has
		// a breakpoint for the program counter only and will break for *any*
		// bank
		check.next = nil
		if err := bp.checkBreaker(check); err != nil {
			return BrkGrpAnyBank
		}
	} else {
		// for cartridges with just one bank a PC break for the
		// disassembly.Entry address is, by definition, a break for *this* bank
		if err := bp.checkBreaker(check); err != nil {
			return BrkGrpThisBank
		}
	}

	// there is no breakpoint at that matches this disassembly entry
	return BrkGrpNone
}
