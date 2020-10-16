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

package bus

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// CPUBus defines the operations for the memory system when accessed from the CPU
// All memory areas implement this interface because they are all accessible
// from the CPU (compare to ChipBus). The VCSMemory type also implements this
// interface and maps the read/write address to the correct memory area --
// meaning that CPU access need not care which part of memory it is writing to
//
// Addresses should be mapped to their primary mirror when accesses the RIOT,
// TIA or RAM; and should be unmapped when accessing cartridge memory (some
// cartridge mappers are sensitive to which cartridge mirror is being used).
type CPUBus interface {
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

type CPUBusZeroPage interface {
	// implementations of ReadZeroPage may just pass the address onto the
	// Read() function and return, depending on what the implementation is
	// supposed to do. for the real vcs emulation however, a zero page read
	// has consequences
	ReadZeroPage(address uint8) (uint8, error)
}

// ChipData packages together the name of the chip register that has been
// written to and the value that was written. Useful for passing values without
// losing context - for example, the UpdateBus.Update() function.
type ChipData struct {
	// the canonical name of the chip register written to
	Name string

	// the data value written to the chip register
	Value uint8
}

// ChipBus defines the operations for the memory system when accessed from the
// VCS chips (TIA, RIOT). Only ChipMemory implements this interface.
type ChipBus interface {
	// ChipRead checks to see if the chip's memory area has been written to. if
	// it has the function returns true and an instance of ChipData
	ChipRead() (bool, ChipData)

	// ChipWrite writes the data to the chip memory
	ChipWrite(reg addresses.ChipRegister, data uint8)

	// LastReadRegister returns the register name of the last memory location
	// *read* by the CPU
	LastReadRegister() string
}

// UpdateBus is a bus internal to the emulation. It exposes the Update()
// function of one sub-system to another sub-system. Currently used to connect
// the RIOT input sub-system to the TIA VBLANK (by calling Update() on the
// input sub-system from the TIA).
type UpdateBus interface {
	Update(ChipData) bool
}

// Sentinal error returned by memory package functions. Note that the error
// expects a numberic address, which will be formatted as four digit hex.
const (
	AddressError = "inaccessible address (%#04x)"
)
