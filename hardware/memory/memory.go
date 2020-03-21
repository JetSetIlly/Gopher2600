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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package memory

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// VCSMemory is the monolithic representation of the memory in 2600.
type VCSMemory struct {
	bus.CPUBus

	// memmap is a hash for every address in the VCS address space, returning
	// one of the four memory areas
	Memmap []bus.DebuggerBus

	// the four memory areas
	RIOT *ChipMemory
	TIA  *ChipMemory
	RAM  *RAM
	Cart *cartridge.Cartridge

	// the following are only used by the debugging interface. it would be
	// lovely to remove these for non-debugging emulation but there's not much
	// impact on performance so they can stay for now:
	//
	//  . a note of the last (mapped) memory address to be accessed
	//  . the value that was written/read from the last address accessed
	//  . whether the last addres accessed was written or read
	//  . the ID of the last memory access (currently a timestamp)
	LastAccessAddress uint16
	LastAccessValue   uint8
	LastAccessWrite   bool
	LastAccessID      int

	// accessCount is incremented every time memory is read or written to.  the
	// current value of accessCount is noted every read and write and
	// immediately incremented.
	//
	// for practical purposes, the cycle period of type int is sufficiently
	// large as to allow us to consider LastAccessID to be unique.
	accessCount int
}

// NewVCSMemory is the preferred method of initialisation for VCSMemory
func NewVCSMemory() (*VCSMemory, error) {
	mem := &VCSMemory{}

	mem.Memmap = make([]bus.DebuggerBus, memorymap.Memtop+1)

	mem.RIOT = newRIOT()
	mem.TIA = newTIA()
	mem.RAM = newRAM()
	mem.Cart = cartridge.NewCartridge()

	if mem.RIOT == nil || mem.TIA == nil || mem.RAM == nil || mem.Cart == nil {
		return nil, errors.New(errors.MemoryError, "cannot create memory areas")
	}

	// create the memory map by associating all addresses in each memory area
	// with that area
	for i := memorymap.OriginTIA; i <= memorymap.MemtopTIA; i++ {
		mem.Memmap[i] = mem.TIA
	}

	for i := memorymap.OriginRAM; i <= memorymap.MemtopRAM; i++ {
		mem.Memmap[i] = mem.RAM
	}

	for i := memorymap.OriginRIOT; i <= memorymap.MemtopRIOT; i++ {
		mem.Memmap[i] = mem.RIOT
	}

	for i := memorymap.OriginCart; i <= memorymap.MemtopCart; i++ {
		mem.Memmap[i] = mem.Cart
	}

	return mem, nil
}

// GetArea returns the actual memory of the specified area type
func (mem *VCSMemory) GetArea(area memorymap.Area) (bus.DebuggerBus, error) {
	switch area {
	case memorymap.TIA:
		return mem.TIA, nil
	case memorymap.RAM:
		return mem.RAM, nil
	case memorymap.RIOT:
		return mem.RIOT, nil
	case memorymap.Cartridge:
		return mem.Cart, nil
	}

	return nil, errors.New(errors.MemoryError, "area not mapped correctly")
}

// read maps an address to the normalised for all memory areas.
func (mem *VCSMemory) read(address uint16, zeroPage bool) (uint8, error) {
	// optimisation: called a lot. pointer to VCSMemory to prevent duffcopy

	ma, ar := memorymap.MapAddress(address, true)
	area, err := mem.GetArea(ar)
	if err != nil {
		return 0, err
	}

	data, err := area.(bus.CPUBus).Read(ma)

	// some memory areas do not change all the bits on the data bus, leaving
	// some bits of the address in the result
	//
	// if the mapped address has an entry in the Mask array then use the most
	// significant byte of the non-mapped address and apply it to the data bits
	// specified in the mask
	//
	// see commentary for DataMasks array for extensive explanation
	if ma < uint16(len(addresses.DataMasks)) {
		if !zeroPage {
			data &= addresses.DataMasks[ma]
			data |= uint8((address>>8)&0xff) & (addresses.DataMasks[ma] ^ 0xff)
		} else {
			data &= addresses.DataMasks[ma]
			data |= uint8(address&0x00ff) & (addresses.DataMasks[ma] ^ 0xff)
		}
	}

	mem.LastAccessAddress = ma
	mem.LastAccessWrite = false
	mem.LastAccessValue = data
	mem.LastAccessID = mem.accessCount
	mem.accessCount++

	return data, err
}

// Read is an implementation of CPUBus. Address will be normalised and
// processed by the correct memory area.
func (mem *VCSMemory) Read(address uint16) (uint8, error) {
	return mem.read(address, false)
}

// ReadZeroPage is an implementation of CPUBus. Address will be normalised and
// processed by the correct memory areas.
func (mem *VCSMemory) ReadZeroPage(address uint8) (uint8, error) {
	return mem.read(uint16(address), true)
}

// Write is an implementation of CPUBus Address will be normalised and
// processed by the correct memory areas.
func (mem *VCSMemory) Write(address uint16, data uint8) error {
	ma, ar := memorymap.MapAddress(address, false)
	area, err := mem.GetArea(ar)
	if err != nil {
		return err
	}

	mem.LastAccessAddress = ma
	mem.LastAccessWrite = true
	mem.LastAccessValue = data
	mem.LastAccessID = mem.accessCount
	mem.accessCount++

	// as incredible as it may seem tigervision cartridges react to memory
	// writes to (unmapped) addresses in the range 0x00 to 0x3f. the Listen()
	// function is a horrible solution to this but I can't see how else to
	// handle it.
	mem.Cart.Listen(address, data)

	return area.(bus.CPUBus).Write(ma, data)
}
