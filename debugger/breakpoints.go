package debugger

import (
	"fmt"
	"headlessVCS/hardware/cpu"
	"strconv"
)

// breakpoints keeps track of all the currently defined breakers and any
// other special conditions that may interrupt execution
type breakpoints struct {
	breaks []breaker
}

// breaker defines a specific break condition
type breaker struct {
	targetName string
	target     breakTarget
	value      uint
}

// breakTarget defines what is and isn't allowed to cause an execution break
type breakTarget interface {
	ToUint() uint
	Size() int
}

// newBreakpoints is the preferred method of initialisation for breakpoins
func newBreakpoints() *breakpoints {
	bp := new(breakpoints)
	bp.breaks = make([]breaker, 0, 10)
	return bp
}

// check compares the current state of the emulation with every break
// condition. it lists every condition that applies, not just the first
// condition it encounters.
func (bp *breakpoints) check(dbg *Debugger, result *cpu.InstructionResult) bool {
	broken := false
	for i := range bp.breaks {
		if bp.breaks[i].target.ToUint() == bp.breaks[i].value {
			dbg.print("break on %s = %s", bp.breaks[i].targetName, bp.breaks[i].valueString())
			broken = true
		}
	}
	return broken
}

func (bp *breakpoints) parseUserInput(dbg *Debugger, parts []string) error {
	if len(parts) == 1 {

		if len(bp.breaks) == 0 {
			dbg.print("no breakpoints\n")
		} else {
			dbg.print("breakpoints\n")
			dbg.print("-----------\n")
			for i := range bp.breaks {
				dbg.print("%s: %s", bp.breaks[i].targetName, bp.breaks[i].valueString())
			}
		}
	}

	var target breakTarget
	target = &dbg.vcs.MC.PC
	targetName := "PC"

	for i := 1; i < len(parts); i++ {
		val, err := strconv.ParseUint(parts[i], 0, 16)
		if err == nil {
			bp.breaks = append(bp.breaks, breaker{targetName: targetName, target: target, value: uint(val)})
		} else {
			switch parts[i] {
			default:
				return fmt.Errorf("unrecognised target (%s) for %s command", parts[i], parts[0])
			case "PC":
				target = &dbg.vcs.MC.PC
				targetName = "PC"
			case "A":
				target = &dbg.vcs.MC.A
				targetName = "A"
			case "X":
				target = &dbg.vcs.MC.X
				targetName = "X"
			case "Y":
				target = &dbg.vcs.MC.Y
				targetName = "Y"
			case "SP":
				target = &dbg.vcs.MC.SP
				targetName = "SP"
			}
		}
	}

	return nil
}

func (bk *breaker) valueString() string {
	if bk.target.Size() == 8 {
		return fmt.Sprintf("%d [0x%02x]\n", bk.value, bk.value)
	}
	return fmt.Sprintf("%d [0x%04x]\n", bk.value, bk.value)
}
