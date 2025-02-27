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

import "fmt"

// SharedMemory represents the memory passed between the parent
// cartridge-mapper implementation and the ARM.
type SharedMemory interface {
	// Return memory block and origin address for the memory block. Memory
	// blocks mays be different for read and write operations.
	//
	// Note that there is no indication of how the memory will be accessed. For
	// instance whether it's for a 32bit or an 8bit access. For this reason the
	// implemention can assume that the access is 8bit and that the user of the
	// result will make further boundary checks as appropriate.
	//
	// The write and executing flags provide context for the MapAddress()
	// implementation. for example, if the executing flag is true then
	// MapAddress() is being called because the ARM will be running instructions
	// stored at that address
	MapAddress(addr uint32, write bool, executing bool) (*[]byte, uint32)

	// Return reset addreses for the Stack Pointer register; the Link Register;
	// and Program Counter
	ResetVectors() (uint32, uint32, uint32)

	// Return true is address contains executable instructions.
	IsExecutable(addr uint32) bool
}

// CartridgeHook is used to extend a cartridge with additional cartridge
// specific functionality
type CartridgeHook interface {
	// ARMinterrupt allows the parent cartridge mapping to emulate ARM code in a
	// more direct way. This is primarily because we do not yet emulate 32bit ARM
	// bytecode
	//
	// Returns false if parent cartridge mapping does not understand the
	// address.
	//
	// * This is primarily here for DPC+ and CDF compatability with the Harmony
	// implementation of those mappers.
	ARMinterrupt(addr uint32, val1 uint32, val2 uint32) (ARMinterruptReturn, error)
}

// Optional extension to the CartridgeHook interface that allows the cartridge
// to annotate an disassembly entry
type CartridgeHookDisassembly interface {
	AnnotateDisassembly(*DisasmEntry) fmt.Stringer
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
