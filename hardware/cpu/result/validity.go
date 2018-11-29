package result

import (
	"fmt"
	"gopher2600/hardware/cpu/definitions"
	"reflect"
)

// IsValid checks whether the instance of StepResult contains consistent data.
//
// Intended to be used during development of the CPU pacakge, to make sure
// implementation hasn't gone off the rails.
func (result Instruction) IsValid() error {
	if !result.Final {
		return fmt.Errorf("not checking an unfinalised InstructionResult: %s", result)
	}

	// check that InstructionData is broadly sensible - is either nil, a uint16 or uint8
	if result.InstructionData != nil {
		ot := reflect.TypeOf(result.InstructionData).Kind()
		if ot != reflect.Uint16 && ot != reflect.Uint8 {
			return fmt.Errorf("instruction data is bad (%s): %s", ot, result)
		}
	}

	// is PageFault valid given content of Defn
	if !result.Defn.PageSensitive && result.PageFault {
		return fmt.Errorf("unexpected page fault: %s", result)
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if result.Bug == "" {
		if result.Defn.AddressingMode == definitions.Relative {
			if result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 && result.ActualCycles != result.Defn.Cycles+2 {
				return fmt.Errorf("number of cycles wrong (%d instead of %d, %d or %d): %s", result.ActualCycles, result.Defn.Cycles, result.Defn.Cycles+1, result.Defn.Cycles+2, result)
			}
		} else {
			if result.Defn.PageSensitive {
				if result.PageFault && result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 {
					fmt.Println(result.Defn)
					return fmt.Errorf("number of cycles wrong (actual %d instead of %d or %d): %s", result.ActualCycles, result.Defn.Cycles, result.Defn.Cycles+1, result)
				}
			} else {
				if result.ActualCycles != result.Defn.Cycles {
					return fmt.Errorf("number of cycles wrong (actual %d instead of %d): %s", result.ActualCycles, result.Defn.Cycles, result)
				}
			}
		}
	}

	return nil
}
