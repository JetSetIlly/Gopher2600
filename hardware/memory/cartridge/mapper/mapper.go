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

package mapper

import (
	"fmt"
	"math/rand"
)

// CartContainer is a special CartMapper type that wraps another CartMapper.
// For example, the PlusROM type.
type CartContainer interface {
	CartMapper
	ContainerID() string
}

// CartMapper implementations hold the actual data from the loaded ROM and
// keeps track of which banks are mapped to individual addresses. for
// convenience, functions with an address argument receive that address
// normalised to a range of 0x0000 to 0x0fff.
type CartMapper interface {
	String() string
	ID() string

	Snapshot() CartSnapshot
	Plumb(CartSnapshot)

	// reset volatile areas of the cartridge. for many cartridge mappers this
	// will do nothing but those with registers or ram should perform an
	// explicit reset (possibly with randomisation)
	Reset(randSrc *rand.Rand)

	Read(addr uint16, active bool) (data uint8, err error)
	Write(addr uint16, data uint8, active bool, poke bool) error
	NumBanks() int
	GetBank(addr uint16) BankInfo

	// see the commentary for the Listen() function in the Cartridge type for
	// an explanation for what this does
	Listen(addr uint16, data uint8)

	// some cartridge mappings have independent clocks that tick and change
	// internal cartridge state. the step() function is called every cpu cycle
	// at a rate of 1.19MHz. cartridges with slower clocks need to handle the
	// rate change.
	Step()

	// patch differs from write/poke in that it alters the data as though it
	// was being read from disk. that is, the offset is measured from the start
	// of the file. the cartmapper must translate the offset and update the
	// correct data structure as appropriate.
	Patch(offset int, data uint8) error

	// return copies of all banks in the cartridge. the disassembly process
	// uses this to access cartridge data freely and without affecting the
	// state of the cartridge.
	CopyBanks() []BankContent
}

// OptionalSuperchip are implemented by cartMappers that have an optional
// superchip. This shouldn't be used to decide if a cartridge has additional
// RAM or not. Use the CartRAMbus interface for that.
type OptionalSuperchip interface {
	AddSuperchip()
}

// CartRegistersBus defines the operations required for a debugger to access the
// registers in a cartridge.
//
// The mapper is allowed to panic if it is not interfaced with correctly.
//
// You should know the precise cartridge mapper for the CartRegisters to be
// usable.
//
// So what's the point of the interface if you need to know the details of the
// underlying type? Couldn't we just use a type assertion?
//
// Yes, but doing it this way helps with the lazy evaluation system used by
// debugging GUIs. The point of the lazy system is to prevent race conditions
// and the way we do that is to make copies of system variables before using it
// in the GUI. Now, because we must know the internals of the cartridge format,
// could we not just make those copies manually? Again, yes. But that would
// mean another place where the cartridge's internal knowledge needs to be
// coded (we need to use that knowledge in the GUI code but it would be nice to
// avoid it in the lazy system).
//
// The GetRegisters() allows us to conceptualise the copying process and to
// keep the details inside the cartridge implementation as much as possible.
type CartRegistersBus interface {
	// GetRegisters returns a copy of the cartridge's registers
	GetRegisters() CartRegisters

	// Update a register in the cartridge with new data.
	//
	// Depending on the complexity of the cartridge, the register argument may
	// need to be a structured string to uniquely identify a register (eg. a
	// JSON string, although that's probably going over the top). The details
	// of what is valid should be specified in the documentation of the mappers
	// that use the CartDebugBus.
	//
	// The data string will be converted to whatever type is required for the
	// register. For simple types then this will be usual Go representation,
	// (eg. true of false for boolean types) but it may be a more complex
	// representation. Again, the details of what is valid should be specified
	// in the mapper documentation.
	PutRegister(register string, data string)
}

// CartRegisters conceptualises the cartridge specific registers that are
// inaccessible through normal addressing.
type CartRegisters interface {
	fmt.Stringer
}

// CartRAMbus is implemented for catridge mappers that have an addressable RAM
// area. This differs from a Static area which is not addressable by the VCS.
//
// Note that for convenience, some mappers will implement this interface but
// have no RAM for the specific cartridge. In these case GetRAM() will return
// nil.
//
// The test for whether a specific cartridge has additional RAM should include
// a interface type asserstion as well as checking GetRAM() == nil.
type CartRAMbus interface {
	GetRAM() []CartRAM
	PutRAM(bank int, idx int, data uint8)
}

// CartRAM represents a single segment of RAM in the cartridge. A cartridge may
// contain more than one segment of RAM. The Label field can help distinguish
// between the different segments.
//
// The Origin field specifies the address of the lowest byte in RAM. The Data
// field is a copy of the actual bytes in the cartidge RAM. Because Cartidge is
// addressable, it is also possible to update cartridge RAM through the normal
// memory buses; although in the context of a debugger it is probably more
// convience to use PutRAM() in the CartRAMbus interface.
type CartRAM struct {
	Label  string
	Origin uint16
	Data   []uint8
	Mapped bool
}

// CartStaticBus defines the operations required for a debugger to access the
// static area of a cartridge.
type CartStaticBus interface {
	// GetStatic returns a copy of the cartridge's static areas
	GetStatic() []CartStatic
	PutStatic(tag string, addr uint16, data uint8) error
}

// CartStatic conceptualises a static data area that is inaccessible through.
// Of the cartridge types that have static areas some have more than one static
// area.
//
// Unlike CartRAM, there is no indication of the origin address of the
// StaticArea. This is because the areas are not addressable in the usual way
// and so the concept of an origin address is meaningless and possibly
// misleading.
type CartStatic struct {
	Label string
	Data  []uint8
}

// CartTapeBus defines additional debugging functions for cartridge types that use tapes.
type CartTapeBus interface {
	// Move tape loading to the specified mark. returns true if rewind was
	// effective
	Rewind() bool

	// Set tape counter to specified value
	SetTapeCounter(c int)

	// GetTapeState retrieves a copy of the current state of the tape. returns
	// true is state is valid
	GetTapeState() (bool, CartTapeState)
}

// CartTapeState is the current state of the tape.
type CartTapeState struct {
	Counter    int
	MaxCounter int
	Time       float64
	MaxTime    float64
	Data       []float32
}

// CartHotspotsBus will be implemented for cartridge mappers that want to report
// details of any special addresses. We'll call these hotspots for all types of
// special addresses, not just bank switches.
//
// The index to the returned maps, must be addresses in the cartridge address
// range. For normality, this should be in the primary cartridge mirror (ie.
// 0x1000 to 0x1fff).
type CartHotspotsBus interface {
	ReadHotspots() map[uint16]CartHotspotInfo
	WriteHotspots() map[uint16]CartHotspotInfo
}

// CartHotspotAction defines the action of a hotspot address.
type CartHotspotAction int

// List of valid CartHotspotActions.
const (
	// the most common type of hotspot is the bankswitch. for these hotspots
	// the bank/segment is switched when the address is read/write.
	HotspotBankSwitch CartHotspotAction = iota

	// some cartridge mappers have additional registers.
	HotspotRegister

	// a function is a catch all category that describes any hotspot address
	// that has some other than or more complex than just bank switching. for
	// example, the Supercharger CONFIG address causes bank-switching to take
	// place but is none-the-less defined as a HotspotFunction.
	HotspotFunction

	// some hotspots will be defined but be unused or reserved by the
	// cartridge.
	HotspotReserved
)

// HotspotInfo details the name and purpose of hotspot address.
type CartHotspotInfo struct {
	Symbol string
	Action CartHotspotAction
}

// CartSnapshot represents saved data from the cartridge as a result of a
// Snapshot() operation.
type CartSnapshot interface {
	// make another copy of the snapshot
	Snapshot() CartSnapshot
}
