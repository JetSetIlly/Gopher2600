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

package memory

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
)

// Memory is the monolithic representation of the memory in 2600.
type Memory struct {
	env *environment.Environment

	RIOT *vcs.RIOTMemory
	TIA  *vcs.TIAMemory
	RAM  *vcs.RAM
	Cart *cartridge.Cartridge

	// the following are only used by the debugging interface
	//
	//  . a note of the last literal memory address to be accessed
	//  . as above but the mapped address
	//  . the value that was written/read from the last address accessed
	//  . whether the last address accessed was written or read
	//
	// * the literal address is the address as it appears in the 6507 program.
	// and therefore it might be more than 13 bits wide. as such it is not
	// representative of what happens on the address bus
	//
	// it is sometimes useful to know what the literal address is, distinct
	// from the mapped address, for debugging purposes.
	LastCPUAddressLiteral uint16
	LastCPUAddressMapped  uint16
	LastCPUData           uint8
	LastCPUWrite          bool

	// the actual values that have been put on the address and data buses.
	AddressBus uint16
	DataBus    uint8

	// not all pins of the databus are driven at all times. bits set in
	// the DataBusDriven field indicate the pins that are being driven
	DataBusDriven uint8
}

// NewMemory is the preferred method of initialisation for Memory.
func NewMemory(env *environment.Environment) *Memory {
	mem := &Memory{
		env:  env,
		RIOT: vcs.NewRIOTMemory(env),
		TIA:  vcs.NewTIAMemory(env),
		RAM:  vcs.NewRAM(env),
		Cart: cartridge.NewCartridge(env),
	}
	mem.Reset()
	return mem
}

func (mem *Memory) String() string {
	return fmt.Sprintf("Address: %016b [%04x]   Data: %08b [%04x]", mem.AddressBus, mem.AddressBus, mem.DataBus, mem.DataBus)
}

// Snapshot creates a copy of the current memory state.
func (mem *Memory) Snapshot() *Memory {
	n := *mem
	n.RIOT = mem.RIOT.Snapshot()
	n.TIA = mem.TIA.Snapshot()
	n.RAM = mem.RAM.Snapshot()
	n.Cart = mem.Cart.Snapshot()
	return &n
}

// Plumb makes sure everything is ship-shape after a rewind event.
//
// The fromDifferentEmulation indicates that the State has been created by a
// different VCS emulation than the one being plumbed into.
func (mem *Memory) Plumb(env *environment.Environment, fromDifferentEmulation bool) {
	mem.env = env
	mem.Cart.Plumb(env, fromDifferentEmulation)
}

// Reset contents of memory.
func (mem *Memory) Reset() {
	mem.RIOT.Reset()
	mem.TIA.Reset()
	mem.RAM.Reset()
	mem.Cart.Reset()
}

// Area defines the meta-operations for all memory areas
type Area interface {
	Read(address uint16) (uint8, uint8, error)
	Write(address uint16, data uint8) error
	Peek(address uint16) (uint8, error)
	Poke(address uint16, value uint8) error
}

// GetArea returns the actual memory of the specified area type.
func (mem *Memory) GetArea(area memorymap.Area) Area {
	switch area {
	case memorymap.TIA:
		return mem.TIA
	case memorymap.RAM:
		return mem.RAM
	case memorymap.RIOT:
		return mem.RIOT
	case memorymap.Cartridge:
		return mem.Cart
	}
	panic("memory areas are not mapped correctly")
}

func (mem *Memory) Read(address uint16) (uint8, error) {
	var err error

	// the address bus value is the literal address masked to the 13 bits
	// available to the 6507
	addressBus := address & memorymap.Memtop

	ma, ar := memorymap.MapAddress(addressBus, true)
	area := mem.GetArea(ar)

	// update address bus if it has changed
	if mem.AddressBus != addressBus {
		mem.AddressBus = addressBus

		// if the address bus has changed then we indicate that to the cartridge
		//
		// note that at this point mem.DataBus has not yet been updated as a
		// result of the read access, so we are effectively calling the function
		// with the "old" data bus
		mem.Cart.AccessPassive(mem.AddressBus, mem.DataBus)
	}

	// if the area is cartridge then we need to consider what happens to
	// cartridge addresses that are volatile. ie. RAM locations and "hotspots".
	//
	// what is happening here is probably not an intentional act by the 6507
	// program but it still needs to be accounted for
	//
	// note that we're using the previous value on the databus. this is because
	// the 6507 is not driving the data bus
	if ar == memorymap.Cartridge {
		err = mem.Cart.Write(mem.AddressBus, mem.DataBus)
	}

	// read data from area. if there is an error, we can just ignore it until
	// we get to the end of the function
	var data uint8
	data, mem.DataBusDriven, err = area.Read(ma)

	// the data bus is not always completely driven. ie. some pins are not powered and are left
	// floating
	//
	// a good example of this are the TIA addresses. see commentary for TIADriverPins for extensive
	// explanation
	if mem.DataBusDriven != 0xff {
		// on a real superchip the pins are more indeterminate. for now,
		// applying an addition random pattern is a good enough emulation for this
		if mem.env != nil && mem.env.Prefs.RandomPins.Get().(bool) {
			data |= uint8(mem.env.Random.Rewindable(0xff)) & (^mem.DataBusDriven)
		} else {
			// this pattern is good for replicating what we see on the pluscart
			// this matches observations made by Al_Nafuur with the following
			// binary:
			//
			// https://atariage.com/forums/topic/329888-indexed-read-page-crossing-and-sc-ram/
			//
			// a different bit pattern can be seen on the Harmony
			//
			// https://atariage.com/forums/topic/285759-stella-getting-into-details-help-wanted/
			data |= mem.LastCPUData & ^mem.DataBusDriven
		}
	}

	// we also need to consider what happens when a cartridge is forcefully
	// driving the data bus
	if stuff, ok := mem.Cart.BusStuff(); ok {
		data = stuff
	}

	// update data bus
	mem.DataBus = data

	// update debugging information
	mem.LastCPUAddressLiteral = address
	mem.LastCPUAddressMapped = ma
	mem.LastCPUWrite = false
	mem.LastCPUData = data

	return data, err
}

func (mem *Memory) Write(address uint16, data uint8) error {
	// the address bus value is the literal address masked to the 13 bits
	// available to the 6507
	addressBus := address & memorymap.Memtop

	ma, ar := memorymap.MapAddress(addressBus, false)
	area := mem.GetArea(ar)

	// drive pins from cartridge
	if stuff, ok := mem.Cart.BusStuff(); ok {
		data = stuff
	}

	// update data bus
	mem.DataBus = data

	// service changes to address bus
	if addressBus != mem.AddressBus {
		mem.AddressBus = addressBus
		err := mem.Cart.AccessPassive(mem.AddressBus, mem.DataBus)
		if err != nil {
			return err
		}
	}

	// update debugging information
	mem.LastCPUAddressLiteral = address
	mem.LastCPUAddressMapped = ma
	mem.LastCPUWrite = true
	mem.LastCPUData = data

	return area.Write(ma, data)
}

func (mem *Memory) Peek(address uint16) (uint8, error) {
	ma, ar := memorymap.MapAddress(address, true)
	return mem.GetArea(ar).Peek(ma)
}

func (mem *Memory) Poke(address uint16, data uint8) error {
	ma, ar := memorymap.MapAddress(address, false)
	return mem.GetArea(ar).Poke(ma, data)
}
