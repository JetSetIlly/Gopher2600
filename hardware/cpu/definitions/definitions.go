package definitions

import "fmt"

// AddressingMode describes the method by which an instruction receives data
// on which to operate
type AddressingMode int

// enumeration of supported addressing modes
const (
	Implied AddressingMode = iota
	Immediate
	Relative // relative addressing is used for branch instructions

	Absolute // sometimes called absolute addressing
	ZeroPage
	Indirect // indirect addressing (with no indexing) is only for JMP instructions

	PreIndexedIndirect  // uses X register
	PostIndexedIndirect // uses Y register
	AbsoluteIndexedX
	AbsoluteIndexedY
	IndexedZeroPageX
	IndexedZeroPageY // only used for LDX
)

// EffectCategory - categorises an instruction by the effect it has
type EffectCategory int

// enumeration of instruction effect categories
const (
	Read EffectCategory = iota
	Write
	RMW

	// the following three effects have a variable effect on the program
	// counter, depending on the instruction's precise operand
	Flow
	Subroutine
	Interrupt
)

// InstructionDefinition - defines each instruction in the instruction set; one per instruction
type InstructionDefinition struct {
	ObjectCode     uint8
	Mnemonic       string
	Bytes          int
	Cycles         int
	AddressingMode AddressingMode
	PageSensitive  bool
	Effect         EffectCategory
}

func (defn InstructionDefinition) String() string {
	if defn.Mnemonic == "" {
		return "undecoded instruction"
	}
	return fmt.Sprintf("%02x %s +%dbytes (%d cycles) [mode=%d pagesens=%t effect=%d]", defn.ObjectCode, defn.Mnemonic, defn.Bytes, defn.Cycles, defn.AddressingMode, defn.PageSensitive, defn.Effect)
}
