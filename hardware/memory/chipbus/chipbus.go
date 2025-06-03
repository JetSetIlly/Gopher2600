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

package chipbus

import "github.com/jetsetilly/gopher2600/hardware/memory/cpubus"

// ChangedRegister packages together the name of the chip register that has been
// changed by the CPU along with the new value
type ChangedRegister struct {
	// the address of the register
	Address uint16

	// the data value written to the chip register
	Value uint8

	// registers are changed via the CPU bus and so the name is a
	// cpubus.Register and not a chipbus.Register
	//
	// of the register is not named then the change is non-effective. the
	// RIOT/TIA implementation should log such an event
	Register cpubus.Register
}

// PokeNotify links a memory area to the chip implementation
type PokeNotify interface {
	AfterPoke(ChangedRegister)
}

// Memory defines the operations for the memory system when accessed from the
// VCS chips (TIA, RIOT)
type Memory interface {
	// SetPokeNotify links a memory area with the chip implementation
	SetPokeNotify(PokeNotify)

	// ChipWrite writes the data to the chip memory
	ChipWrite(reg Register, data uint8)

	// ChipRefer reads the data from chip memory. It returns the value that the
	// CPU will see.
	//
	// Should be used in preference to keeping a local copy of the written
	// value in the TIA/RIOT implementation.
	ChipRefer(reg Register) uint8

	// LastReadAddress returns true and the address of the last read by the
	// CPU. Returns false if no read has taken place since the last call to the
	// function.
	//
	// Only used by the RIOT timer.
	LastReadAddress() (bool, uint16)
}

// Register specifies the offset of a chip register in the chip memory areas. It
// is used in contexts where a register is required, as opposed to an address
type Register int

// TIA registers
//
// These value are used by the emulator to specify known addresses. For
// example, when writing collision information we know we need the CXM0P
// register. these named values make the code more readable
const (
	CXM0P Register = iota
	CXM1P
	CXP0FB
	CXP1FB
	CXM0FB
	CXM1FB
	CXBLPF
	CXPPMM
	INPT0
	INPT1
	INPT2
	INPT3
	INPT4
	INPT5
)

// RIOT registers
//
// These value are used by the emulator to specify known addresses. For
// example, the timer updates itself every cycle and stores time remaining
// value in the INTIM register
const (
	SWCHA Register = iota
	SWACNT
	SWCHB
	SWBCNT
	INTIM
	TIMINT
)
