package result

import (
	"gopher2600/hardware/cpu/instructions"
)

// Instruction contains all the interesting information about a CPU
// instruction. Including the address it was read from, the data that was used
// during the execution of the instruction and a reference to the instruction
// definition.
//
// Other fields record inforamtion about the last execution of this specific
// instruction, whether it caused a page fault, the actual number of cycles it
// took and any information about known bugs that might have been triggered in
// the CPU.
//
// The Instruction type is update every cycle during execution in the emulated
// CPU. As more information is known about the instuction it is added. The
// final field indicates whether the last cycle has been executed and the
// instruction decoding is complete.
type Instruction struct {
	Address uint16

	// it would be lovely to have a note of which cartridge bank the address is
	// in, but we want to keep the 6507 emulation as non-specific as possible.
	// if you need to know the cartridge bank then you need to get it somehow
	// else.

	// a reference to the instruction definition
	Defn *instructions.Definition

	// instruction data is the actual instruction data. so, for example, in the
	// case of ranch instruction, instruction data is the offset value.
	InstructionData interface{}

	// whether this data has been finalised - some fields in this struct will
	// be undefined if Final is false
	Final bool

	// the actual number of cycles taken by the instruction - usually the same
	// as Defn.Cycles but in the case of PageFaults and branches, this value
	// may be different
	ActualCycles int

	// whether an extra cycle was required because of 8 bit adder overflow
	PageFault bool

	// whether a known buggy code path (in the emulated CPU) was triggered
	Bug string
}
