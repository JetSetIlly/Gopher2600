package cpu

func (mc *CPU) drainCycles() (*InstructionResult, error) {
	var res *InstructionResult
	var err error
	for mc.IsRunning() {
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
