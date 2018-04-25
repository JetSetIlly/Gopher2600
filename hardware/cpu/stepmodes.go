package cpu

// drainCycles is used to align cpu execution to the next instruction point.
// this is a belt & braces function - ideally, the body of the for loop will
// never run - but switching between StepCycle and StepInstruction in a
// debugger may require it
func (mc *CPU) drainCycles() (*InstructionResult, error) {
	var res *InstructionResult
	var err error
	for mc.IsExecutingInstruction() {
		res, err = mc.StepCycle()
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

// StepInstruction executes the next instruction in the program
func (mc *CPU) StepInstruction() (*InstructionResult, error) {
	// drain previous instruction if it is mid cycle
	res, err := mc.drainCycles()
	if err != nil || res != nil {
		return res, err
	}

	for {
		mc.stepNext <- true

		select {
		case result := <-mc.stepResult:
			if result.Final {
				return result, nil
			}
		case err := <-mc.stepError:
			return nil, err
		}
	}
}

// StepCycle runs the next cycle in an instruction, starting a new instruction
// if necessary
func (mc *CPU) StepCycle() (*InstructionResult, error) {
	mc.stepNext <- true

	select {
	case result := <-mc.stepResult:
		return result, nil
	case err := <-mc.stepError:
		return nil, err
	}
}
