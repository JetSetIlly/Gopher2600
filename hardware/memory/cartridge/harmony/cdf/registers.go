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
	"fmt"
	"strconv"
	"strings"

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
}

// Registers implements mappers.Registers.
type Registers struct {
	MusicFetcher [3]musicDataFetcher
	FastFetch    bool
	SampleMode   bool

	Datastream [8]datastream
}

func (r *Registers) initialise() {
	for i := range r.MusicFetcher {
		r.MusicFetcher[i].Waveform = 0x1b
	}
	r.FastFetch = false
	r.SampleMode = false
}

func (r Registers) String() string {
	s := strings.Builder{}
	return s.String()
}

// GetRegisters implements the bus.CartDebugBus interface.
func (cart *cdf) GetRegisters() mapper.CartRegisters {
	for i := range cart.state.registers.Datastream {
		cart.state.registers.Datastream[i].Pointer = cart.readDatastreamPointer(i)
		cart.state.registers.Datastream[i].Increment = cart.readDatastreamIncrement(i)
	}

	return cart.state.registers
}

// PutRegister implements the bus.CartDebugBus interface
//
// Register specification is divided with the "::" string. The following table
// describes what the valid register strings and, after the = sign, the type to
// which the data argument will be converted.
//
//	datastream::%int::pointer = uint8
//	datastream::%int::increment = uint8
//	music::%int::waveform = uint8
//	music::%int::freq = uint8
//	music::%int::count = uint8
//	fastfetch = bool
//	samplemode = bool
//
// note that PutRegister() will panic() if the register or data string is invalid.
func (cart *cdf) PutRegister(register string, data string) {
	d32, _ := strconv.ParseUint(data, 16, 32)
	d8, _ := strconv.ParseUint(data, 16, 8)

	r := strings.Split(register, "::")
	switch r[0] {
	case "datastream":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.Datastream) {
			panic(fmt.Sprintf("cdf: unrecognised register [%s]", register))
		}
		switch r[2] {
		case "pointer":
			cart.updateDatastreamPointer(f, uint32(d32))
		case "increment":
			cart.updateDatastreamIncrement(f, uint32(d32))
		}
	case "music":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.Datastream) {
			panic(fmt.Sprintf("cdf: unrecognised register [%s]", register))
		}
		switch r[2] {
		case "waveform":
			cart.state.registers.MusicFetcher[f].Waveform = uint8(d8)
		case "freq":
			cart.state.registers.MusicFetcher[f].Freq = uint32(d8)
		case "increment":
			cart.state.registers.MusicFetcher[f].Count = uint32(d8)
		default:
			panic(fmt.Sprintf("cdf: unrecognised register [%s]", register))
		}
	case "fastfetch":
		switch data {
		case "true":
			cart.state.registers.FastFetch = true
		case "false":
			cart.state.registers.FastFetch = false
		default:
			panic(fmt.Sprintf("cdf: unrecognised boolean state [%s]", data))
		}
	case "samplemode":
		switch data {
		case "true":
			cart.state.registers.SampleMode = true
		case "false":
			cart.state.registers.SampleMode = false
		default:
			panic(fmt.Sprintf("cdf: unrecognised boolean state [%s]", data))
		}
	default:
		panic(fmt.Sprintf("cdf: unrecognised variable [%s]", register))
	}
}
