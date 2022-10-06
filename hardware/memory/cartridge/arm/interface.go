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

package arm

// SharedMemory represents the memory passed between the parent
// cartridge-mapper implementation and the ARM.
type SharedMemory interface {
	// Return memory block and array offset for the requested address. Memory
	// blocks mays be different for read and write operations.
	MapAddress(addr uint32, write bool) (*[]byte, uint32)

	// Return reset addreses for the Stack Pointer register; the Link Register;
	// and Program Counter
	ResetVectors() (uint32, uint32, uint32)

	// Return true is address contains executable instructions.
	IsExecutable(addr uint32) bool
}

// CartridgeHook allows the parent cartridge mapping to emulate ARM code in a
// more direct way. This is primarily because we do not yet emulate full ARM
// bytecode only Thumb bytecode, and the value of doing so is unclear.
type CartridgeHook interface {
	// Returns false if parent cartridge mapping does not understand the
	// address.
	ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (ARMinterruptReturn, error)
}

// ARMInterruptReturn is the return value of the ARMinterrupt type.
type ARMinterruptReturn struct {
	InterruptEvent      string
	SaveResult          bool
	SaveRegister        uint32
	SaveValue           uint32
	InterruptServiced   bool
	NumMemAccess        int
	NumAdditionalCycles int
}
