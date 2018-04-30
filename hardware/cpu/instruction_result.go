package cpu

import (
	"fmt"
	"gopher2600/hardware/cpu/definitions"
	"reflect"
)

// InstructionResult contains all the interesting information from a CPU step.
type InstructionResult struct {
	ProgramCounter  uint16
	Defn            definitions.InstructionDefinition
	InstructionData interface{}
	ActualCycles    int

	// whether an extra cycle was required because of 8 bit adder overflow
	PageFault bool

	// whether a buggy (CPU) code path was triggered
	Bug string

	// whether this data has been finalised
	Final bool
}

func (result InstructionResult) String() string {
	var data string
	var pf, bug string

	if !result.Final {
		return "*"
	}

	if result.Defn.Bytes == 2 {
		data = fmt.Sprintf("$%02x", result.InstructionData)
	} else if result.Defn.Bytes == 3 {
		data = fmt.Sprintf("$%04x", result.InstructionData)
	}

	switch result.Defn.AddressingMode {
	case definitions.Implied:
	case definitions.Immediate:
		data = fmt.Sprintf("#%s", data)
	case definitions.Relative:
	case definitions.Absolute:
	case definitions.ZeroPage:
	case definitions.Indirect:
		data = fmt.Sprintf("(%s)", data)
	case definitions.PreIndexedIndirect:
		data = fmt.Sprintf("(%s,X)", data)
	case definitions.PostIndexedIndirect:
		data = fmt.Sprintf("(%s),Y", data)
	case definitions.AbsoluteIndexedX:
		data = fmt.Sprintf("%s,X", data)
	case definitions.AbsoluteIndexedY:
		data = fmt.Sprintf("%s,Y", data)
	case definitions.IndexedZeroPageX:
		data = fmt.Sprintf("%s,X", data)
	case definitions.IndexedZeroPageY:
		data = fmt.Sprintf("%s,Y", data)
	default:
	}

	if result.PageFault {
		pf = " page-fault"
	}

	if result.Bug != "" {
		bug = fmt.Sprintf(" * %s *", result.Bug)
	}

	return fmt.Sprintf("0x%04x\t%s\t%s\t[%d]%s%s", result.ProgramCounter, result.Defn.Mnemonic, data, result.ActualCycles, pf, bug)
}

// IsValid checks whether the instance of StepResult contains consistent data.
//
// Intended to be used during development of the CPU pacakge, to make sure
// implementation hasn't gone off the rails.
func (result InstructionResult) IsValid() error {
	if !result.Final {
		return fmt.Errorf("not checking an unfinalised InstructionResult")
	}

	// check that InstructionData is broadly sensible - is either nil, a uint16 or uint8
	if result.InstructionData != nil {
		ot := reflect.TypeOf(result.InstructionData).Kind()
		if ot != reflect.Uint16 && ot != reflect.Uint8 {
			return fmt.Errorf("instruction data is bad (%s)", ot)
		}
	}

	// is PageFault valid given content of Defn
	if !result.Defn.PageSensitive && result.PageFault {
		return fmt.Errorf("unexpected page fault")
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if result.Bug != "" {
		if result.Defn.AddressingMode == definitions.Relative {
			if result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 && result.ActualCycles != result.Defn.Cycles+2 {
				return fmt.Errorf("number of cycles wrong (%d instead of %d, %d or %d)", result.ActualCycles, result.Defn.Cycles, result.Defn.Cycles+1, result.Defn.Cycles+2)
			}
		} else {
			if result.Defn.PageSensitive {
				if result.PageFault && result.ActualCycles != result.Defn.Cycles && result.ActualCycles != result.Defn.Cycles+1 {
					return fmt.Errorf("number of cycles wrong (%d instead of %d or %d)", result.ActualCycles, result.Defn.Cycles, result.Defn.Cycles+1)
				}
			} else {
				if result.ActualCycles != result.Defn.Cycles {
					return fmt.Errorf("number of cycles wrong (%d instead of %d", result.ActualCycles, result.Defn.Cycles)
				}
			}
		}
	}

	return nil
}
