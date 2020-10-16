// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package execution

import (
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
)

// Result records the state/result of each instruction executed on the CPU.
// Including the address it was read from, a reference to the instruction
// definition, and other execution details.
//
// The Result type is updated every cycle during the execution of the emulated
// CPU. As the execution continues, more information is acquired and detail
// added to the Result.
//
// The Final field indicates whether the last cycle of the instruction has been
// executed. An instance of Result with a Final value of false can still be
// used but with the caveat that the information is incomplete. Note that a
// Defn of nil means the opcode hasn't even been decoded.
type Result struct {
	// a reference to the instruction definition
	Defn *instructions.Definition

	// the number of bytes read during instruction decode. if this value is
	// less than Defn.Bytes then the instruction has not yet been fully decoded
	ByteCount int

	// the address at which the instruction began
	Address uint16

	// it would be lovely to have a note of which cartridge bank the address is
	// in, but we want to keep the 6507 emulation as non-specific as possible.
	// if you need to know the cartridge bank then you need to get it somehow
	// else.

	// instruction data is the actual instruction data. so, for example, in the
	// case of branch instruction, instruction data is the offset value.
	InstructionData uint16

	// the actual number of cycles taken by the instruction - usually the same
	// as Defn.Cycles but in the case of PageFaults and branches, this value
	// may be different
	Cycles int

	// whether an extra cycle was required because of 8 bit adder overflow
	PageFault bool

	// whether a known buggy code path (in the emulated CPU) was triggered
	CPUBug string

	// error string. will be a memory access error
	Error string

	// whether branch instruction test passed (ie. branched) or not. testing of
	// this field should be used in conjunction with Defn.IsBranch()
	BranchSuccess bool

	// whether this data has been finalised - some fields in this struct will
	// be undefined if Final is false
	Final bool
}

// Reset nullifies all members of the Result instance.
func (r *Result) Reset() {
	r.Defn = nil
	r.ByteCount = 0
	r.Address = 0
	r.InstructionData = 0
	r.Cycles = 0
	r.PageFault = false
	r.CPUBug = ""
	r.Error = ""
	r.Final = false
}
