package debugger

import (
	"fmt"
	"headlessVCS/hardware"
	"headlessVCS/hardware/cpu"
	"strconv"
)

type breakpoints struct {
	pc map[uint16]bool
}

func newBreakpoints() *breakpoints {
	bp := new(breakpoints)
	bp.pc = make(map[uint16]bool)
	return bp
}

func (bp *breakpoints) check(vcs *hardware.VCS, result *cpu.InstructionResult) bool {
	v, _ := bp.pc[vcs.MC.PC.ToUint16()]
	return v
}

func (bp *breakpoints) add(parts []string) error {
	if len(parts) < 2 {
		return fmt.Errorf("not enough arguments for BREAK command")
	}
	i, err := strconv.ParseUint(parts[1], 0, 16)
	if err != nil {
		return fmt.Errorf("cannot convert argument to BREAK command")
	}
	bp.pc[uint16(i)] = true

	return nil
}
