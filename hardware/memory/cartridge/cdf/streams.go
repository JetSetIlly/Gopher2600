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
func (ds datastream) Peek(y int, mem []uint8) uint8 {
	p := ds.AfterCALLFN
	p += (ds.Increment << ds.incrementShift) * uint32(y)

	if int(p>>ds.fetcherShift) >= len(mem) {
		return 0
	}

	return mem[p>>ds.fetcherShift]
}

func (cart *cdf) readDatastreamPointer(reg int) uint32 {
	idx := cart.version.fetcherBase + (uint32(reg) * 4)
	return uint32(cart.state.static.driverRAM[idx]) |
		uint32(cart.state.static.driverRAM[idx+1])<<8 |
		uint32(cart.state.static.driverRAM[idx+2])<<16 |
		uint32(cart.state.static.driverRAM[idx+3])<<24
}

func (cart *cdf) readDatastreamIncrement(inc int) uint32 {
	idx := cart.version.incrementBase + (uint32(inc) * 4)
	return uint32(cart.state.static.driverRAM[idx]) |
		uint32(cart.state.static.driverRAM[idx+1])<<8 |
		uint32(cart.state.static.driverRAM[idx+2])<<16 |
		uint32(cart.state.static.driverRAM[idx+3])<<24
}

func (cart *cdf) updateDatastreamPointer(reg int, data uint32) {
	if reg < len(cart.state.registers.Datastream) {
		cart.state.registers.Datastream[reg].Pointer = data
	}

	idx := cart.version.fetcherBase + (uint32(reg) * 4)
	cart.state.static.driverRAM[idx] = uint8(data)
	cart.state.static.driverRAM[idx+1] = uint8(data >> 8)
	cart.state.static.driverRAM[idx+2] = uint8(data >> 16)
	cart.state.static.driverRAM[idx+3] = uint8(data >> 24)
}

// updateDatastreamIncrement is not used by the CDF mapper itself except as a
// call from PutRegister(), which is a debugging facility.
func (cart *cdf) updateDatastreamIncrement(reg int, data uint32) {
	if reg < len(cart.state.registers.Datastream) {
		cart.state.registers.Datastream[reg].Increment = data
	}

	idx := cart.version.incrementBase + (uint32(reg) * 4)
	cart.state.static.driverRAM[idx] = uint8(data)
	cart.state.static.driverRAM[idx+1] = uint8(data >> 8)
	cart.state.static.driverRAM[idx+2] = uint8(data >> 16)
	cart.state.static.driverRAM[idx+3] = uint8(data >> 24)
}

func (cart *cdf) readMusicFetcher(mus int) uint32 {
	// CDFJ+ differences ??

	addr := cart.version.musicBase + (uint32(mus) * 4)
	return uint32(cart.state.static.driverRAM[addr]) |
		uint32(cart.state.static.driverRAM[addr+1])<<8 |
		uint32(cart.state.static.driverRAM[addr+2])<<16 |
		uint32(cart.state.static.driverRAM[addr+3])<<24
}

func (cart *cdf) streamData(reg int) uint8 {
	addr := cart.readDatastreamPointer(reg)
	inc := cart.readDatastreamIncrement(reg)

	value := cart.state.static.dataRAM[addr>>cart.version.fetcherShift]
	addr += inc << cart.version.incrementShift
	cart.updateDatastreamPointer(reg, addr)

	return value
}
