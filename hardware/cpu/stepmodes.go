package cpu

import (
	"fmt"
)

// StepInstruction executes the next instruction in the program
func (mc *CPU) StepInstruction() (*InstructionResult, error) {
	if mc.IsRunning() {
		return nil, fmt.Errorf("instruction is already running")
	}

	go mc.executeInstruction()

	for {
		mc.stepNext <- true

		select {
		case result := <-mc.stepResult:
			if result.Final {
				return &result, nil
			}
		case err := <-mc.stepError:
			return nil, err
		}
	}
}

// StepCycle runs the next cycle in an instruction, starting a new instruction
// if necessary
func (mc *CPU) StepCycle() (*InstructionResult, error) {
	if !mc.IsRunning() {
		go mc.executeInstruction()
	}

	mc.stepNext <- true

	select {
	case result := <-mc.stepResult:
		return &result, nil
	case err := <-mc.stepError:
		return nil, err
	}
}
