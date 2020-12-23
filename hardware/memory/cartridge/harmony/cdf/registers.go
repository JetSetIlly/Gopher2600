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

// Registers implements mappers.Registers
type Registers struct {
	MusicFetcher [3]musicDataFetcher
	FastFetch    bool
	SampleMode   bool
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

type musicDataFetcher struct {
	Waveform uint8
	Freq     uint32
	Count    uint32
}

// GetRegisters implements the bus.CartDebugBus interface.
func (cart *cdf) GetRegisters() mapper.CartRegisters {
	return cart.state.registers
}

func (cart *cdf) PutRegister(register string, data string) {
	// most data is expected to an integer (a uint8 specifically) so we try
	// to convert it here. if it doesn't convert then it doesn't matter
	d, _ := strconv.ParseUint(data, 16, 8)

	r := strings.Split(register, "::")
	switch r[0] {
	case "music":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.MusicFetcher) {
			panic(fmt.Sprintf("unrecognised fetcher [%s]", register))
		}
		switch r[2] {
		case "waveform":
			cart.state.registers.MusicFetcher[f].Waveform = uint8(d)
		case "freq":
			cart.state.registers.MusicFetcher[f].Freq = uint32(d)
		case "increment":
			cart.state.registers.MusicFetcher[f].Count = uint32(d)
		default:
			panic(fmt.Sprintf("unrecognised variable [%s]", register))
		}
	case "fastfetch":
		switch data {
		case "true":
			cart.state.registers.FastFetch = true
		case "false":
			cart.state.registers.FastFetch = false
		default:
			panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
		}
	case "samplemode":
		switch data {
		case "true":
			cart.state.registers.SampleMode = true
		case "false":
			cart.state.registers.SampleMode = false
		default:
			panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
		}
	default:
		panic(fmt.Sprintf("unrecognised variable [%s]", register))
	}
}
