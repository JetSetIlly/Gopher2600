package cpu

import (
	"fmt"
	"reflect"
)

// StepResult contains all the interesting information from a CPU step.
type StepResult struct {
	ProgramCounter  uint16
	Defn            InstructionDefinition
	InstructionData interface{}
	ActualCycles    int

	// whether an extra cycle was required because of 8 bit adder overflow
	PageFault bool

	// whether a buggy (CPU) code path was triggered
	Bug string
}

func (sr StepResult) String() string {
	var data string
	var pf, bug string

	if sr.Defn.Bytes == 2 {
		data = fmt.Sprintf("$%02x", sr.InstructionData)
	} else if sr.Defn.Bytes == 3 {
		data = fmt.Sprintf("$%04x", sr.InstructionData)
	}

	switch sr.Defn.AddressingMode {
	case Implied:
	case Immediate:
		data = fmt.Sprintf("#%s", data)
	case Relative:
	case Absolute:
	case ZeroPage:
	case Indirect:
		data = fmt.Sprintf("(%s)", data)
	case PreIndexedIndirect:
		data = fmt.Sprintf("(%s,X)", data)
	case PostIndexedIndirect:
		data = fmt.Sprintf("(%s),Y", data)
	case AbsoluteIndexedX:
		data = fmt.Sprintf("%s,X", data)
	case AbsoluteIndexedY:
		data = fmt.Sprintf("%s,Y", data)
	case IndexedZeroPageX:
		data = fmt.Sprintf("%s,X", data)
	case IndexedZeroPageY:
		data = fmt.Sprintf("%s,Y", data)
	default:
	}

	if sr.PageFault {
		pf = " page-fault"
	}

	if sr.Bug != "" {
		bug = fmt.Sprintf(" * %s *", sr.Bug)
	}

	return fmt.Sprintf("0x%04x\t%s\t%s\t[%d]%s%s", sr.ProgramCounter, sr.Defn.Mnemonic, data, sr.ActualCycles, pf, bug)
}

// IsValid checks whether the instance of StepResult contains consistent data.
//
// Intended to be used during development of the CPU pacakge, to make sure
// implementation hasn't gone off the rails.
func (sr StepResult) IsValid() error {
	// check that InstructionData is broadly sensible - is either nil, a uint16 or uint8
	if sr.InstructionData != nil {
		ot := reflect.TypeOf(sr.InstructionData).Kind()
		if ot != reflect.Uint16 && ot != reflect.Uint8 {
			return fmt.Errorf("instruction data is bad (%s)", ot)
		}
	}

	// is PageFault valid given content of Defn
	if !sr.Defn.PageSensitive && sr.PageFault {
		return fmt.Errorf("unexpected page fault")
	}

	// if a bug has been triggered, don't perform the number of cycles check
	if sr.Bug != "" {
		if sr.Defn.AddressingMode == Relative {
			if sr.ActualCycles != sr.Defn.Cycles && sr.ActualCycles != sr.Defn.Cycles+1 {
				return fmt.Errorf("number of cycles wrong (%d instead of %d or %d)", sr.ActualCycles, sr.Defn.Cycles, sr.Defn.Cycles+1)
			}
		} else {
			if sr.Defn.PageSensitive {
				if sr.PageFault && sr.ActualCycles != sr.Defn.Cycles && sr.ActualCycles != sr.Defn.Cycles+1 {
					return fmt.Errorf("number of cycles wrong (%d instead of %d or %d)", sr.ActualCycles, sr.Defn.Cycles, sr.Defn.Cycles+1)
				}
			} else {
				if sr.ActualCycles != sr.Defn.Cycles {
					return fmt.Errorf("number of cycles wrong (%d instead of %d", sr.ActualCycles, sr.Defn.Cycles)
				}
			}
		}
	}

	return nil
}
