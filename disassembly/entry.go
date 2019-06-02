package disassembly

import (
	"gopher2600/hardware/cpu/definitions"
	"gopher2600/hardware/cpu/result"
)

// Entry for every address in the cartridge
type Entry struct {
	// if the type of entry is, or appears to be a, a valid instruction then
	// instructionDefinition will be non-null
	instructionDefinition *definitions.InstructionDefinition

	// to keep things simple, we're only keeping a string representation of the
	// disassembly. we used to keep a instance of result.verbose but after
	// some consideration, I don't like that - the result was obtained with an
	// incomplete or misleading context, and so cannot be relied upon to give
	// accurate information with regards to pagefaults etc. the simplest way
	// around this is to record a string representation, containing only the
	// information that's required.
	//
	// undefined if InstructionDefinition == nil
	instruction string

	// the styling used to format the instruction member above
	style result.Style
}

func (ent Entry) String() string {
	if ent.instructionDefinition == nil {
		return ""
	}
	return ent.instruction
}

// IsInstruction returns false if the entry does not represent an instruction
func (ent Entry) IsInstruction() bool {
	return ent.instructionDefinition != nil
}
