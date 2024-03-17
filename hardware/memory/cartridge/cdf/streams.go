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

package cdf

import (
	"encoding/binary"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

type musicDataFetcher struct {
	Waveform uint8
	Freq     uint32
	Count    uint32
}

type datastream struct {
	Pointer   uint32
	Increment uint32

	// the value of Pointer immediately after the most recent CALLFN. the
	// Pointer field is updated after every Fetch, this field is not
	AfterCALLFN uint32

	// the Peek function requires knowledge of the incrementShift and
	// fetcherShift values for the format. these values can change depending on
	// the precise CDF version. these are copies of the values in the version
	// type
	incrementShift uint32
	fetcherShift   uint32
}

// Peek returns the value at the Nth increment of the base pointer. Useful for
// predicting or peeking at what the Nth value of a stream will be.
func (ds datastream) Peek(y int, mem mapper.CartStatic) uint8 {
	m := mem.(*Static).dataRAM.data

	p := ds.AfterCALLFN
	p += (ds.Increment << ds.incrementShift) * uint32(y)

	if int(p>>ds.fetcherShift) >= len(m) {
		return 0
	}

	return m[p>>ds.fetcherShift]
}

func (cart *cdf) readDatastreamPointer(reg int) uint32 {
	idx := cart.version.fetcherBase + (uint32(reg) * 4)
	return binary.LittleEndian.Uint32(cart.state.static.driverRAM.data[idx:])
}

func (cart *cdf) readDatastreamIncrement(inc int) uint32 {
	idx := cart.version.incrementBase + (uint32(inc) * 4)
	return binary.LittleEndian.Uint32(cart.state.static.driverRAM.data[idx:])
}

func (cart *cdf) updateDatastreamPointer(reg int, data uint32) {
	if reg < len(cart.state.registers.Datastream) {
		cart.state.registers.Datastream[reg].Pointer = data
	}

	idx := cart.version.fetcherBase + (uint32(reg) * 4)
	binary.LittleEndian.PutUint32(cart.state.static.driverRAM.data[idx:], data)
}

// updateDatastreamIncrement is not used by the CDF mapper itself except as a
// call from PutRegister(), which is a debugging facility.
func (cart *cdf) updateDatastreamIncrement(reg int, data uint32) {
	if reg < len(cart.state.registers.Datastream) {
		cart.state.registers.Datastream[reg].Increment = data
	}

	idx := cart.version.incrementBase + (uint32(reg) * 4)
	binary.LittleEndian.PutUint32(cart.state.static.driverRAM.data[idx:], data)
}

func (cart *cdf) readMusicFetcher(mus int) uint32 {
	// CDFJ+ differences ??

	addr := cart.version.musicBase + (uint32(mus) * 4)
	return binary.LittleEndian.Uint32(cart.state.static.driverRAM.data[addr:])
}

func (cart *cdf) streamData(reg int) uint8 {
	addr := cart.readDatastreamPointer(reg)
	inc := cart.readDatastreamIncrement(reg)

	idx := int(addr >> cart.version.fetcherShift)
	if idx >= len(cart.state.static.dataRAM.data) {
		return 0
	}
	value := cart.state.static.dataRAM.data[idx]

	addr += inc << cart.version.incrementShift
	cart.updateDatastreamPointer(reg, addr)

	return value
}
