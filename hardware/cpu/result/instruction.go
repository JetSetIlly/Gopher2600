package result

import (
	"fmt"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/register"
	"gopher2600/symbols"
	"reflect"
)

// Instruction contains all the interesting information from a CPU step.
type Instruction struct {
	Address         uint16
	Defn            definitions.InstructionDefinition
	InstructionData interface{}
	ActualCycles    int

	// whether an extra cycle was required because of 8 bit adder overflow
	PageFault bool

	// whether a known buggy code path (int the emulated CPU) was triggered
	Bug string

	// whether this data has been finalised
	Final bool
}

func (result Instruction) String() string {
	return result.GetString(nil, 0)
}

// GetString returns a human readable version of InstructionResult, addresses
// replaced with symbols if supplied symbols argument is not null. prefer this
// function to implicit calls to String()
func (result Instruction) GetString(symtable *symbols.Table, style Style) string {
	// columns
	var hex string
	var label, programCounter string
	var operator, operand string
	var notes string

	// include instruction address (and label) if this is the final result for
	// this particular instruction
	if result.Final {
		programCounter = fmt.Sprintf("0x%04x", result.Address)
		if symtable != nil && style.Has(StyleFlagLocation) {
			if v, ok := symtable.Locations[result.Address]; ok {
				label = v
			}
		}
	}

	// use mnemonic if specified in instruciton result
	if result.Defn.Mnemonic == "" {
		operator = "???"
	} else {
		operator = result.Defn.Mnemonic
	}

	// parse instruction result data ...
	var idx uint16
	switch result.InstructionData.(type) {
	case uint8:
		idx = uint16(result.InstructionData.(uint8))
		operand = fmt.Sprintf("$%02x", idx)
	case uint16:
		idx = uint16(result.InstructionData.(uint16))
		operand = fmt.Sprintf("$%04x", idx)
	case nil:
		if result.Defn.Bytes == 2 {
			operand = "??"
		} else if result.Defn.Bytes == 3 {
			operand = "????"
		}
	}

	if result.Final && style.Has(StyleFlagHex) {
		switch result.Defn.Bytes {
		case 3:
			hex = fmt.Sprintf("%02x", idx&0xff00>>8)
			fallthrough
		case 2:
			hex = fmt.Sprintf("%02x %s", idx&0x00ff, hex)
			fallthrough
		case 1:
			hex = fmt.Sprintf("%02x %s", result.Defn.ObjectCode, hex)
		default:
			hex = fmt.Sprintf("(%d bytes) %s", result.Defn.Bytes, hex)
		}
	}

	// ... and use assembler symbol for the operand if available/appropriate
	if symtable.Valid && style.Has(StyleFlagSymbols) && result.InstructionData != nil && (operand == "" || operand[0] != '?') {
		if result.Defn.AddressingMode != definitions.Immediate {

			switch result.Defn.Effect {
			case definitions.Flow:
				if result.Defn.AddressingMode == definitions.Relative {
					// relative labels. to get the correct label we have to
					// simulate what a successful branch instruction would do:

					// 	-- we create a mock register with the instruction's
					// 	address as the initial value
					pc := register.NewAnonRegister(result.Address, 16)

					// -- add the number of instruction bytes to get the PC as
					// it would be at the end of the instruction
					pc.Add(uint8(result.Defn.Bytes), false)

					// -- because we're doing 16 bit arithmetic with an 8bit
					// value, we need to make sure the sign bit has been
					// propogated to the more-significant bits
					if idx&0x0080 == 0x0080 {
						idx |= 0xff00
					}

					// -- add the 2s-complement value to the mock program
					// counter
					pc.Add(idx, false)

					// -- look up mock program counter value in symbol table
					if v, ok := symtable.Locations[pc.ToUint16()]; ok {
						operand = v
					}

				} else {
					if v, ok := symtable.Locations[idx]; ok {
						operand = v
					}
				}
			case definitions.Read:
				if v, ok := symtable.ReadSymbols[idx]; ok {
					operand = v
				}
			case definitions.Write:
				fallthrough
			case definitions.RMW:
				if v, ok := symtable.WriteSymbols[idx]; ok {
					operand = v
				}
			}
		}
	}

	// decorate operand with addressing mode indicators
	switch result.Defn.AddressingMode {
	case definitions.Implied:
	case definitions.Immediate:
		operand = fmt.Sprintf("#%s", operand)
	case definitions.Relative:
	case definitions.Absolute:
	case definitions.ZeroPage:
	case definitions.Indirect:
		operand = fmt.Sprintf("(%s)", operand)
	case definitions.PreIndexedIndirect:
		operand = fmt.Sprintf("(%s,X)", operand)
	case definitions.PostIndexedIndirect:
		operand = fmt.Sprintf("(%s),Y", operand)
	case definitions.AbsoluteIndexedX:
		operand = fmt.Sprintf("%s,X", operand)
	case definitions.AbsoluteIndexedY:
		operand = fmt.Sprintf("%s,Y", operand)
	case definitions.IndexedZeroPageX:
		operand = fmt.Sprintf("%s,X", operand)
	case definitions.IndexedZeroPageY:
		operand = fmt.Sprintf("%s,Y", operand)
	default:
	}

	// cycles annotation depends on whether the result is in its final form
	if style.Has(StyleFlagNotes) {
		if result.Final {
			notes = fmt.Sprintf("[%d]", result.ActualCycles)
		} else {
			notes = "[v]"
		}

		// add annotation for page-faults and known CPU bugs
		if result.PageFault {
			notes += " page-fault"
		}
		if result.Bug != "" {
			notes += fmt.Sprintf(" * %s *", result.Bug)
		}
	}

	// force column widths
	if style.Has(StyleFlagColumns) {
		if style.Has(StyleFlagHex) {
			hex = columnise(hex, 8)
		}
		programCounter = columnise(programCounter, 6)
		operator = columnise(operator, 3)
		if symtable.Valid {
			label = columnise(label, symtable.MaxLocationWidth)

			// +3 to MaxSymbolWidth so that additional notation (parenthesis,
			// etc.) isn't cropped off.
			operand = columnise(operand, symtable.MaxSymbolWidth+3)
		} else {
			label = columnise(label, 0)
			operand = columnise(operand, 7)
		}
	}

	// build final string
	s := fmt.Sprintf("%s %s %s %s %s %s",
		hex,
		label,
		programCounter,
		operator,
		operand,
		notes)

	return s
}

// IsValid checks whether the instance of StepResult contains consistent data.
//
// Intended to be used during development of the CPU pacakge, to make sure
// implementation hasn't gone off the rails.
func (result Instruction) IsValid() error {
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
