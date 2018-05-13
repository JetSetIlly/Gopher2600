package debugger

import (
	"fmt"
	"gopher2600/hardware/cpu"
	"strconv"
)

// breakpoints keeps track of all the currently defined breakers and any
// other special conditions that may interrupt execution
type breakpoints struct {
	breaks []breaker
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
func newBreakpoints() *breakpoints {
	bp := new(breakpoints)
	bp.clear()
	return bp
}

func (bp *breakpoints) clear() {
	bp.breaks = make([]breaker, 0, 10)
}

// check compares the current state of the emulation with every break
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (bp *breakpoints) check(dbg *Debugger, result *cpu.InstructionResult) bool {
	broken := false
	for i := range bp.breaks {
		if bp.breaks[i].target.ToInt() == bp.breaks[i].value {
			dbg.print(Feedback, "break on %v", bp.breaks[i].valueString())
			broken = true
		}
	}
	return broken
}

func (bp *breakpoints) parseBreakpoint(dbg *Debugger, parts []string) error {
	if len(parts) == 1 {

		if len(bp.breaks) == 0 {
			dbg.print(Feedback, "no breakpoints")
		} else {
			dbg.print(Feedback, "breakpoints")
			dbg.print(Feedback, "-----------")
			for i := range bp.breaks {
				dbg.print(Feedback, "%s", bp.breaks[i].valueString())
			}
		}
	}

	var target breakTarget
	target = dbg.vcs.MC.PC

	for i := 1; i < len(parts); i++ {
		val, err := strconv.ParseUint(parts[i], 0, 16)
		if err == nil {
			bp.breaks = append(bp.breaks, breaker{target: target, value: int(val)})
		} else {

			// TODO: namespaces so we can do things like "BREAK TV COLOR RED" without
			// our breakpoints code knowing anything about it. GetTVState() will
			// return a TVState if the television implementation understands the
			// request

			switch parts[i] {
			default:
				return fmt.Errorf("unrecognised target (%s) for %s command", parts[i], parts[0])
			case "PC":
				target = dbg.vcs.MC.PC
			case "A":
				target = dbg.vcs.MC.A
			case "X":
				target = dbg.vcs.MC.X
			case "Y":
				target = dbg.vcs.MC.Y
			case "SP":
				target = dbg.vcs.MC.SP
			case "FRAMENUM", "FRAME", "FR":
				target, err = dbg.vcs.TV.GetTVState("FRAMENUM")
				if err != nil {
					return err
				}
			case "SCANLINE", "SL":
				target, err = dbg.vcs.TV.GetTVState("SCANLINE")
				if err != nil {
					return err
				}
			case "HORIZPOS", "HP":
				target, err = dbg.vcs.TV.GetTVState("HORIZPOS")
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
