package result

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu/definitions"
	"reflect"
)

// IsValid checks whether the instance of StepResult contains consistent data.
//
// Intended to be used during development of the CPU pacakge, to make sure
// implementation hasn't gone off the rails.
func (result Instruction) IsValid() error {
	if !result.Final {
		return errors.NewFormattedError(errors.InvalidResult, "not checking an unfinalised InstructionResult", result)
	}

	// check that InstructionData is broadly sensible - is either nil, a uint16 or uint8
	if result.InstructionData != nil {
		ot := reflect.TypeOf(result.InstructionData).Kind()
		if ot != reflect.Uint16 && ot != reflect.Uint8 {
			return errors.NewFormattedError(errors.InvalidResult, fmt.Sprintf("instruction data is bad (%s)", ot), result)
		}
	}

	// is PageFault valid given content of Defn
	if !result.Defn.PageSensitive && result.PageFault {
		return errors.NewFormattedError(errors.InvalidResult, "unexpected page fault", result)
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if result.Bug == "" {
		if result.Defn.AddressingMode == definitions.Relative {
			if result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 && result.ActualCycles != result.Defn.Cycles+2 {
				msg := fmt.Sprintf("number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d or %d)",
					result.Defn.ObjectCode,
					result.Defn.Mnemonic,
					result.ActualCycles,
					result.Defn.Cycles,
					result.Defn.Cycles+1,
					result.Defn.Cycles+2)
				return errors.NewFormattedError(errors.InvalidResult, msg)
			}
		} else {
			if result.Defn.PageSensitive {
				if result.PageFault && result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 {
					msg := fmt.Sprintf("number of cycles wrong for opcode %#02x [%s] (%d instead of %d, %d)",
						result.Defn.ObjectCode,
						result.Defn.Mnemonic,
						result.ActualCycles,
						result.Defn.Cycles,
						result.Defn.Cycles+1)
					return errors.NewFormattedError(errors.InvalidResult, msg)
				}
			} else {
				if result.ActualCycles != result.Defn.Cycles {
					msg := fmt.Sprintf("number of cycles wrong for opcode %#02x [%s] (%d instead of %d)",
						result.Defn.ObjectCode,
						result.Defn.Mnemonic,
						result.ActualCycles,
						result.Defn.Cycles)
					return errors.NewFormattedError(errors.InvalidResult, msg)
				}
			}
		}
	}

	return nil
}
