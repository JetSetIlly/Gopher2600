package result

import (
	"fmt"
	"gopher2600/hardware/cpu/instructions"
	"gopher2600/hardware/cpu/registers"
	"gopher2600/symbols"
	"strings"
)

// GetString returns a human readable version of InstructionResult, addresses
// replaced with symbols if supplied symbols argument is not null.
func (result Instruction) GetString(symtable *symbols.Table, style Style) string {
	if symtable == nil {
		panic(fmt.Sprintf("Instruction.GetString() requires a non-nil instance of symbols.Table"))
	}

	// columns
	var hex string
	var label, programCounter string
	var operator, operand string
	var notes string

	// include instruction address (and label) if this is the final result for
	// this particular instruction
	if result.Final && style.Has(StyleFlagAddress) {
		programCounter = fmt.Sprintf("0x%04x", result.Address)
		if style.Has(StyleFlagLocation) {
			if v, ok := symtable.Locations.Symbols[result.Address]; ok {
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
				hex = fmt.Sprintf("%02x %s", result.Defn.OpCode, hex)
			default:
				hex = fmt.Sprintf("(%d bytes) %s", result.Defn.Bytes, hex)
			}
		}

		// ... and use assembler symbol for the operand if available/appropriate
		if style.Has(StyleFlagSymbols) && result.InstructionData != nil && (operand == "" || operand[0] != '?') {
			if result.Defn.AddressingMode != instructions.Immediate {

				switch result.Defn.Effect {
				case instructions.Flow:
					if result.Defn.AddressingMode == instructions.Relative {
						// relative labels. to get the correct label we have to
						// simulate what a successful branch instruction would do:

						// 	-- we create a mock register with the instruction's
						// 	address as the initial value
						pc := registers.NewProgramCounter(result.Address)

						// -- add the number of instruction bytes to get the PC as
						// it would be at the end of the instruction
						pc.Add(uint16(result.Defn.Bytes))

						// -- because we're doing 16 bit arithmetic with an 8bit
						// value, we need to make sure the sign bit has been
						// propogated to the more-significant bits
						if idx&0x0080 == 0x0080 {
							idx |= 0xff00
						}

						// -- add the 2s-complement value to the mock program
						// counter
						pc.Add(idx)

						// -- look up mock program counter value in symbol table
						if v, ok := symtable.Locations.Symbols[pc.Address()]; ok {
							operand = v
						}

					} else {
						if v, ok := symtable.Locations.Symbols[idx]; ok {
							operand = v
						}
					}
				case instructions.Read:
					if v, ok := symtable.Read.Symbols[idx]; ok {
						operand = v
					}
				case instructions.Write:
					fallthrough
				case instructions.RMW:
					if v, ok := symtable.Write.Symbols[idx]; ok {
						operand = v
					}
				}
			}
		}

		// decorate operand with addressing mode indicators
		switch result.Defn.AddressingMode {
		case instructions.Implied:
		case instructions.Immediate:
			operand = fmt.Sprintf("#%s", operand)
		case instructions.Relative:
		case instructions.Absolute:
		case instructions.ZeroPage:
		case instructions.Indirect:
			operand = fmt.Sprintf("(%s)", operand)
		case instructions.PreIndexedIndirect:
			operand = fmt.Sprintf("(%s,X)", operand)
		case instructions.PostIndexedIndirect:
			operand = fmt.Sprintf("(%s),Y", operand)
		case instructions.AbsoluteIndexedX:
			operand = fmt.Sprintf("%s,X", operand)
		case instructions.AbsoluteIndexedY:
			operand = fmt.Sprintf("%s,Y", operand)
		case instructions.IndexedZeroPageX:
			operand = fmt.Sprintf("%s,X", operand)
		case instructions.IndexedZeroPageY:
			operand = fmt.Sprintf("%s,Y", operand)
		default:
		}
	}

	if style.Has(StyleFlagCycles) {
		if result.Final {
			// result is of a complete instruction - add number of cycles it
			// actually took to execute
			notes = fmt.Sprintf("[%d]", result.ActualCycles)
		} else {
			// result is an interim result - indicate with [v], which means
			// video cycle
			notes = "[v]"
		}
	}

	// add annotation
	if style.Has(StyleFlagNotes) {
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
	s := fmt.Sprintf("%s %s %s %s %s %s",
		hex,
		label,
		programCounter,
		operator,
		operand,
		notes)

	if style.Has(StyleFlagCompact) {
		return strings.Trim(s, " ")
	}

	return s
}

// columnise forces the string into the given width. used for outputting
// disassembly into columns
func columnise(s string, width int) string {
	if width > len(s) {
		t := make([]byte, width-len(s))
		for i := 0; i < len(t); i++ {
			t[i] = ' '
		}
		s = fmt.Sprintf("%s%s", s, t)
	} else if width < len(s) {
		s = s[:width]
	}
	return s
}
