package debugger

import (
	"fmt"
	"gopher2600/hardware/cpu"
	"strconv"
)

// breakpoints keeps track of all the currently defined breakers and any
// other special conditions that may interrupt execution
type breakpoints struct {
	dbg               *Debugger
	breaks            []breaker
	storedBreakStates map[breakTarget]int
}

// breaker defines a specific break condition
type breaker struct {
	target breakTarget
	value  int
}

// breakTarget defines what objects can and cannot cause an execution break.
// known implementations of breakTarget:
//  1. register
//  2. tvstate
type breakTarget interface {
	AsString(interface{}) string
	ToInt() int
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

// storeBreakState stores the current value of all current break targets
func (bp *breakpoints) storeBreakState() {
	bp.storedBreakStates = make(map[breakTarget]int, len(bp.breaks))
	for _, b := range bp.breaks {
		bp.storedBreakStates[b.target] = b.target.ToInt()
	}
}

// check compares the current state of the emulation with every break
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (bp *breakpoints) check(dbg *Debugger, result *cpu.InstructionResult) bool {
	broken := false
	for i := range bp.breaks {
		if bp.breaks[i].target.ToInt() == bp.breaks[i].value {
			// make sure that we're not breaking on a state already broken upon
			if bp.breaks[i].target.ToInt() != bp.storedBreakStates[bp.breaks[i].target] {
				dbg.print(Feedback, "break on %v", bp.breaks[i].valueString())
				broken = true
			}
		}
	}
	return broken
}

func (bp breakpoints) list() {
	if len(bp.breaks) == 0 {
		bp.dbg.print(Feedback, "no breakpoints")
	} else {
		for i := range bp.breaks {
			bp.dbg.print(Feedback, "%s", bp.breaks[i].valueString())
		}
	}
}

func (bp *breakpoints) parseBreakpoint(parts []string) error {
	if len(parts) == 1 {
		bp.list()
	}

	var target breakTarget

	// default target of CPU PC. meaning that "BREAK n" will cause a breakpoint
	// being set on the PC. breaking on PC is probably the most common type of
	// breakpoint. the target will change value when the input string sees
	// something appropriate
	target = bp.dbg.vcs.MC.PC

	// loop over parts. if part is a number then add the breakpoint for the
	// current target. if it is not a number, look for a command ro try to change
	// the target (or run a BREAK meta-command)
	//
	// note that this method of looping allows the user to chain break commands
	for i := 1; i < len(parts); i++ {

		val, err := strconv.ParseUint(parts[i], 0, 16)
		if err == nil {
			// check to see if breakpoint already exists
			addNewBreak := true
			for _, mv := range bp.breaks {
				if mv.target == target && mv.value == int(val) {
					addNewBreak = false
					bp.dbg.print(Feedback, "breakpoint (%s) already exists", target.AsString(int(val)))
					break // for loop
				}
			}
			if addNewBreak {
				bp.breaks = append(bp.breaks, breaker{target: target, value: int(val)})
			}

		} else {

			// TODO: namespaces so we can do things like "BREAK TV COLOR RED" without
			// our breakpoints code knowing anything about it. GetTVState() will
			// return a TVState if the television implementation understands the
			// request

			switch parts[i] {
			default:
				return fmt.Errorf("invalid %s target (%s)", parts[0], parts[i])

				// comands
			case "CLEAR":
				bp.clear()
				bp.dbg.print(Feedback, "breakpoints cleared")
			case "LIST":
				bp.list()

				// targets
			case "PC":
				target = bp.dbg.vcs.MC.PC
			case "A":
				target = bp.dbg.vcs.MC.A
			case "X":
				target = bp.dbg.vcs.MC.X
			case "Y":
				target = bp.dbg.vcs.MC.Y
			case "SP":
				target = bp.dbg.vcs.MC.SP
			case "FRAMENUM", "FRAME", "FR":
				target, err = bp.dbg.vcs.TV.GetTVState("FRAMENUM")
				if err != nil {
					return err
				}
			case "SCANLINE", "SL":
				target, err = bp.dbg.vcs.TV.GetTVState("SCANLINE")
				if err != nil {
					return err
				}
			case "HORIZPOS", "HP":
				target, err = bp.dbg.vcs.TV.GetTVState("HORIZPOS")
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (bk *breaker) valueString() string {
	return bk.target.AsString(bk.value)
}
