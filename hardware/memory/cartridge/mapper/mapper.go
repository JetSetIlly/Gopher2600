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
	"io"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/riot/timer"
)

// CartContainer is a special CartMapper type that wraps another CartMapper.
// For example, the PlusROM type.
type CartContainer interface {
	CartMapper
	ContainerID() string
}

// CartDrivenPins is included for clarity. In the vast majority of cases a cartridge mapper
// will drive all pins on the data bus during access. Use CartDrivenPins rather than 0xff.
//
// In the case where the data bus pins are not driven a 0 will suffice.
//
// For any cases where the databus is partially driven, the appropriate value can be used.
const CartDrivenPins = 0xff

// CartMapper implementations hold the actual data from the loaded ROM and
// keeps track of which banks are mapped to individual addresses. for
// convenience, functions with an address argument receive that address
// normalised to a range of 0x0000 to 0x0fff.
type CartMapper interface {
	MappedBanks() string
	ID() string

	Snapshot() CartMapper
	Plumb(*environment.Environment)

	// reset volatile areas of the cartridge. for many cartridge mappers this
	// will do nothing but those with registers or ram should perform an
	// explicit reset (possibly with randomisation)
	Reset()

	// access the cartridge at the specified address. the cartridge is expected to
	// drive the data bus and so this can be thought of as a "read" operation
	//
	// the mask return value allows the mapper to identify which data pins
	// which are being driven by the cartridge. in most cases, the mask should
	// be the CartDrivenPins value
	//
	// the address parameter should be normalised. ie. no mirror information
	Access(addr uint16, peek bool) (data uint8, mask uint8, err error)

	// access the cartridge at the specified volatile address. if the location
	// at that address is not volatile, the data value can be ignored
	//
	// we can think of this as a write operation but it will be called during a
	// read operation too. this is because the cartidge cannot distinguish read
	// and write operations and any access of a volatile address will affect it
	//
	// the address parameter should be normalised. ie. no mirror information
	AccessVolatile(addr uint16, data uint8, poke bool) error

	NumBanks() int
	GetBank(addr uint16) BankInfo

	// AccessPassive is called so that the cartridge can respond to changes to the
	// address and data bus even when the data bus is not addressed to the cartridge.
	//
	// see the commentary for the AccessPassive() function in the Cartridge type
	// for an explanation for why this is needed
	AccessPassive(addr uint16, data uint8) error

	// some cartridge mappings have independent clocks that tick and change
	// internal cartridge state. the step() function is called every cpu cycle
	// at the rate specified.
	Step(clock float32)

	// return copies of all banks in the cartridge. the disassembly process
	// uses this to access cartridge data freely and without affecting the
	// state of the cartridge.
	CopyBanks() []BankContent
}

// SelectableBank is implemented by mappers that can have the selected bank
// changed explicitely by the emulation
type SelectableBank interface {
	SetBank(bank string) error
}

// TerminalCommand allows a mapper to react to terminal commands from the
// debugger
type TerminalCommand interface {
	ParseCommand(w io.Writer, command string) error
}

// PlumbFromDifferentEmulation is for mappers that are sensitive to being
// transferred from one emulation to another.
//
// When state is being plumbed into a different emulation to the one that has
// been created then this interface should be used when available, instead of
// the normal Plumb().
type PlumbFromDifferentEmulation interface {
	PlumbFromDifferentEmulation(*environment.Environment)
}

// OptionalSuperchip are implemented by CartMapper implementations that require
// an optional superchip. This shouldn't be used to decide if a cartridge has
// additional RAM or not. Use the CartRAMbus interface for that.
type OptionalSuperchip interface {
	// the force argument causes the superchip to be added whether it needs it or not
	AddSuperchip(force bool)
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

	// Update the value at the index of the specified RAM bank. Note that this
	// is not the address; it refers to the Data array as returned by GetRAM()
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

// CartRegistersBus defines the operations required for a debugger to access
// any coprocessor in a cartridge.
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
	// that use the CartRegistersbus.
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
	String() string
}

// CartStaticBus defines the operations required for a debugger to access the
// static memory of a cartridge.
//
// Static memory is so called because it is inaccessible from the 6507 program.
// From that point of view the memory is static and can't be changed. It may
// however be changed by any coprocessor on the cartridge.
//
// (Historically, the StaticBus and related types were added to support the DPC
// mapper type, where the memory indeed never can change. When later cartridge
// types (DPC+ and CDF)  were added, the name stuck).
type CartStaticBus interface {
	// GetStatic returns a copy of the cartridge's static areas
	GetStatic() CartStatic

	// ReferenceStatic returns the cartridge's live static areas. Be careful with goroutines
	ReferenceStatic() CartStatic

	// Update the value at the index of the specified segment. the segment
	// argument should come from the Name field of the CartStaticSegment type
	// returned by CartStatic.Segments()
	//
	// The idx field should count from 0 and be no higher than the size of
	// memory in the segment (the differenc of Memtop and Origin returned in
	// the CartStaticSegment type).
	//
	// Returns false if segment is unknown or idx is out of range.
	//
	// PutStatic() will be working on the original data so PutStatic() should be
	// run in the same goroutine as the main emulation.
	PutStatic(segment string, idx int, data uint8) bool
}

// CartStaticSegment describes a single region of the underlying CartStatic
// memory. The Name field can be used to reference the actual memory or to
// update the underlying memory with CartStaticBus.PutStatic()
type CartStaticSegment struct {
	Name   string
	Origin uint32
	Memtop uint32
}

// CartStatic conceptualises a static data area that is inaccessible through the 6507.
type CartStatic interface {
	// returns a list of memory areas in the cartridge's static memory
	Segments() []CartStaticSegment

	// returns a copy of the data in the named segment. the segment name should
	// be taken from the Name field of one of the CartStaticSegment instances
	// returned by the Segments() function
	Reference(segment string) ([]uint8, bool)

	// read 8, 16 or 32 bit values from the address. the address should be in
	// the range given in one of the CartStaticSegment returned by the
	// Segments() function.
	Read8bit(addr uint32) (uint8, bool)
	Read16bit(addr uint32) (uint16, bool)
	Read32bit(addr uint32) (uint32, bool)

	Write8bit(addr uint32, data uint8) bool
}

// CartTapeBus defines additional debugging functions for cartridge types that use tapes.
type CartTapeBus interface {
	// Move tape loading to the beginning of the tape
	Rewind()

	// Set tape counter to specified value
	SetTapeCounter(c int)

	// GetTapeState retrieves a copy of the current state of the tape. returns
	// false if state is not valid.
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

// CartSuperChargerFastLoad defines the commit function required when loading
// Supercharger 'Fastload' binaries
type CartSuperChargerFastLoad interface {
	Fastload(mc *cpu.CPU, ram *vcs.RAM, tmr *timer.Timer) error
}

// CartLabelsBus will be implemented for cartridge mappers that want to report any
// special labels for the cartridge type.
type CartLabelsBus interface {
	Labels() CartLabels
}

// CartLabels is returned by CartLabelsBus. Maps addresses to symbols. Address
// can be any address, not just those in the cartridge.
//
// Currently, addresses are specific and should not be mirrored.
type CartLabels map[uint16]string

// CartLabelsBus and CartHotspotsBus may be combined in the future into a
// single CartSymbolsBus.

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

// CartRewindBoundary is implemented by cartridge mappers that require special
// handling from the rewind system. For some cartridge types it is not
// appropriate to allow rewind history to survive past a certain point.
type CartRewindBoundary interface {
	RewindBoundary() bool
}

// CartROMDump is implemented by cartridge mappers that can save themselves to disk.
type CartROMDump interface {
	ROMDump(filename string) error
}

// CartBusStuff is implemented by cartridge mappers than can arbitrarily drive
// the pins on the data bus during a write.
type CartBusStuff interface {
	BusStuff() (uint8, bool)
}

// CartPatchable is implemented by cartridge mappers than can have their binary
// patched as part of the load process
type CartPatchable interface {
	// patch is different to poke in that it alters the data as though it was
	// being read from disk. that is, the offset is measured from the start of
	// the file. the cartmapper must translate the offset and update the correct
	// data structure as appropriate.
	Patch(offset int, data uint8) error
}
