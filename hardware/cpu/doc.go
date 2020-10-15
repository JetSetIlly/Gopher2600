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

// Package cpu emulates the 6507 microprocessor found in the Atari VCS. Like
// all 8-bit processors of the era, the 6507 executes instructions according to
// the single byte value read from an address pointed to by the program
// counter. This single byte is the opcode and is looked up in the instruction
// table. The instruction definition for that opcode is then used to move
// execution of the program forward.
//
// The instance of the CPU type require an instance of a bus.CPUBus
// implementation as the sole argument. The CPUBus interface defines the memory
// operations required by the CPU. See the bus package for details.
//
// The bread-and-butter of the CPU type is the ExecuteInstruction() function.
// Its sole argument is a callback function to be called at every cycle boundary
// of the instruction.
//
// Let's assume mem is an instance of the CPUBus interface loaded 6507
// instructions.
//
//	mc, _ := cpu.NewCPU(mem)
//
//	numCycles := 0
//	numInstructions := 0
//
//	for {
//		mc.ExecuteInstruction(func() error {
//			numCycles ++
//		})
//		numInstructions ++
//	}
//
// The above program does nothing interesting except to show how
// ExecuteInstruction() can be used to pump information to an callback
// function. The VCS emulation uses this to run the TIA emulation three times
// for every CPU cycle - the CPU clock runs at 1.19MHz while the TIA clock runs
// at 3.57Mhz. TIA emulation is discussed more fully in the TIA package.
//
// The CPU type contains some public fields that are worthy of mention. The
// LastResult field can be probed for information about the last instruction
// executed, or about the current instruction being executed if accessed from
// ExecuteInstruction()'s callback function. See the result package for more
// information. Very useful for debuggers.
//
// The NoFlowControl flag is used by the disassembly package to prevent the CPU
// from honouring "flow control" functions (ie. JMP, BNE, BEQ, etc.). See
// instructions package for classifications.
package cpu
