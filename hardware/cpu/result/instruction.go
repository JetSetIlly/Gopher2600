package result

import (
	"fmt"
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/register"
	"gopher2600/symbols"
)

// Instruction contains all the interesting information from a CPU step.
type Instruction struct {
	Address uint16
	Defn    *definitions.InstructionDefinition

	// whether this data has been finalised - note that the values of the
	// following ields in this struct maybe undefined unless Final is true
	Final bool

	// instruction data is the actual instruction data. so, for example, in the
	// case of a branch instruction, it is the offset value.
	InstructionData interface{}

	// the actual number of cycles taken by the instruction - usually the same
	// as Defn.Cycles but in the case of PageFaults and branches, this value
	// may be different
	ActualCycles int

	// whether an extra cycle was required because of 8 bit adder overflow
	PageFault bool

	// whether a known buggy code path (int the emulated CPU) was triggered
	Bug string
}

func (result Instruction) String() string {
	return result.GetString(nil, StyleBrief)
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

	// use question marks where instruction hasn't been decoded yet

	if result.Defn == nil {
		// nothing has been decoded yet
		operator = "???"

	} else {
		// use mnemonic if specified in instruciton result
		operator = result.Defn.Mnemonic

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

		// (include byte code in output)
		if result.Final && style.Has(StyleFlagByteCode) {
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
		if style.Has(StyleFlagSymbols) && result.InstructionData != nil && (operand == "" || operand[0] != '?') {
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
	}

	// add annotation
	if style.Has(StyleFlagNotes) {
		if result.Final {
			// result is of a complete instruction - add number of cycles it
			// actually took to execute
			notes = fmt.Sprintf("[%d]", result.ActualCycles)
		} else {
			// result is an interim result - indicate with [v], which means
			// video cycle
			notes = "[v]"
		}

		// add annotation for page-faults and known CPU bugs - these can occur
		// whether or not the result is not yet 'final'
		if result.PageFault {
			notes += " page-fault"
		}
		if result.Bug != "" {
			notes += fmt.Sprintf(" * %s *", result.Bug)
		}
	}

	// force column widths
	if style.Has(StyleFlagColumns) {
		if style.Has(StyleFlagByteCode) {
			hex = columnise(hex, 8)
		}
		programCounter = columnise(programCounter, 6)
		operator = columnise(operator, 3)
		if symtable.MaxLocationWidth > 0 {
			label = columnise(label, symtable.MaxLocationWidth)
		} else {
			label = columnise(label, 0)
		}

		if symtable.MaxSymbolWidth > 0 {
			// +3 to MaxSymbolWidth so that additional notation (parenthesis,
			// etc.) isn't cropped off.
			operand = columnise(operand, symtable.MaxSymbolWidth+3)
		} else {
			operand = columnise(operand, 7)
		}
	}

	// build final string
	return fmt.Sprintf("%s %s %s %s %s %s",
		hex,
		label,
		programCounter,
		operator,
		operand,
		notes)
}
