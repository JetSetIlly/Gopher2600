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

package harmony

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
)

// DPCplusRegisters implements the bus.CartRegisters interface.
type DPCplusRegisters struct {
	Fetcher      [8]dataFetcher
	FracFetcher  [8]fractionalDataFetcher
	MusicFetcher [3]musicDataFetcher

	// random number generator
	RNG randomNumberFetcher

	// fast fetch read mode
	FastFetch bool
}

func (r DPCplusRegisters) String() string {
	s := strings.Builder{}

	s.WriteString(fmt.Sprintf("RNG: %#08x\n", r.RNG.Value))
	s.WriteString(fmt.Sprintf("Fast Fetch: %#v\n", r.FastFetch))

	s.WriteString("\nData Fetchers\n")
	s.WriteString("-------------\n")
	for f := 0; f < len(r.Fetcher); f++ {
		s.WriteString(fmt.Sprintf("F%d: l:%#02x h:%#02x t:%#02x b:%#02x", f,
			r.Fetcher[f].Low,
			r.Fetcher[f].Hi,
			r.Fetcher[f].Top,
			r.Fetcher[f].Bottom,
		))
		s.WriteString("\n")
	}

	s.WriteString("\nFractional Data Fetchers\n")
	s.WriteString("------------------------\n")
	for f := 0; f < len(r.FracFetcher); f++ {
		s.WriteString(fmt.Sprintf("F%d: l:%#02x h:%#02x i:%#02x c:%#02x", f,
			r.FracFetcher[f].Low,
			r.FracFetcher[f].Hi,
			r.FracFetcher[f].Increment,
			r.FracFetcher[f].Count,
		))
		s.WriteString("\n")
	}

	s.WriteString("\nMusic Fetchers\n")
	s.WriteString("--------------\n")
	for f := 0; f < len(r.MusicFetcher); f++ {
		s.WriteString(fmt.Sprintf("F%d: w:%#02x f:%#02x c:%#02x", f,
			r.MusicFetcher[f].Waveform,
			r.MusicFetcher[f].Freq,
			r.MusicFetcher[f].Count,
		))
		s.WriteString("\n")
	}

	return s.String()
}

func (r *DPCplusRegisters) reset(randSrc *rand.Rand) {
	for i := range r.Fetcher {
		if randSrc != nil {
			r.Fetcher[i].Low = byte(randSrc.Intn(0xff))
			r.Fetcher[i].Hi = byte(randSrc.Intn(0xff))
			r.Fetcher[i].Top = byte(randSrc.Intn(0xff))
			r.Fetcher[i].Bottom = byte(randSrc.Intn(0xff))
		} else {
			r.Fetcher[i].Low = 0
			r.Fetcher[i].Hi = 0
			r.Fetcher[i].Top = 0
			r.Fetcher[i].Bottom = 0
		}
	}

	for i := range r.FracFetcher {
		if randSrc != nil {
			r.FracFetcher[i].Low = byte(randSrc.Intn(0xff))
			r.FracFetcher[i].Hi = byte(randSrc.Intn(0xff))
			r.FracFetcher[i].Increment = byte(randSrc.Intn(0xff))
			r.FracFetcher[i].Count = byte(randSrc.Intn(0xff))
		} else {
			r.FracFetcher[i].Low = 0
			r.FracFetcher[i].Hi = 0
			r.FracFetcher[i].Increment = 0
			r.FracFetcher[i].Count = 0
		}
	}

	for i := range r.MusicFetcher {
		if randSrc != nil {
			r.MusicFetcher[i].Waveform = uint32(randSrc.Intn(0xffffffff))
			r.MusicFetcher[i].Freq = uint32(randSrc.Intn(0xffffffff))
			r.MusicFetcher[i].Count = uint32(randSrc.Intn(0xffffffff))
		} else {
			r.MusicFetcher[i].Waveform = 0
			r.MusicFetcher[i].Freq = 0
			r.MusicFetcher[i].Count = 0
		}
	}

	if randSrc != nil {
		r.RNG.Value = uint32(randSrc.Intn(0xffffffff))
	} else {
		r.RNG.Value = 0
	}
}

type dataFetcher struct {
	Low byte
	Hi  byte

	Top    byte
	Bottom byte
}

type fractionalDataFetcher struct {
	Low byte
	Hi  byte

	Increment byte
	Count     byte
}

type musicDataFetcher struct {
	Waveform uint32
	Freq     uint32
	Count    uint32
}

type randomNumberFetcher struct {
	Value uint32
}

func (df *dataFetcher) isWindow() bool {
	// unlike the original DPC format checing to see if a data fetcher is in
	// its window has to be done on demand. it has to be like this because the
	// demo ROMs that show off the DPC+ format require it. to put it simply, if
	// we implemented the window flag is it is described in the DPC patent then
	// the DPC+ demo ROMs would miss the window by setting the low attribute
	// toa high (ie. beyond the top value) for the window to caught in the
	// flag->true condition.

	if df.Top > df.Bottom {
		return df.Low > df.Top || df.Low < df.Bottom
	}
	return df.Low > df.Top && df.Low < df.Bottom
}

func (df *dataFetcher) inc() {
	df.Low++
	if df.Low == 0x00 {
		df.Hi++
	}
}

func (df *dataFetcher) dec() {
	df.Low--
	if df.Low == 0xff {
		df.Hi--
	}
}

func (df *fractionalDataFetcher) inc() {
	df.Count += df.Increment
	if df.Count < df.Increment {
		df.Low++
		if df.Low == 0x00 {
			df.Hi++
		}
	}
}

func (rng *randomNumberFetcher) next() {
	if rng.Value&(1<<10) != 0 {
		rng.Value = 0x10adab1e ^ ((rng.Value >> 11) | (rng.Value << 21))
	} else {
		rng.Value = 0x00 ^ ((rng.Value >> 11) | (rng.Value << 21))
	}
}

func (rng *randomNumberFetcher) prev() {
	if rng.Value&(1<<31) != 0 {
		rng.Value = ((0x10adab1e & rng.Value) << 11) | ((0x10adab1e ^ rng.Value) >> 21)
	} else {
		rng.Value = (rng.Value << 11) | (rng.Value >> 21)
	}
}

// GetRegisters implements the bus.CartDebugBus interface.
func (cart dpcPlus) GetRegisters() mapper.CartRegisters {
	return cart.state.registers
}

// PutRegister implements the bus.CartDebugBus interface
//
// Register specification is divided with the "::" string. The following table
// describes what the valid register strings and, after the = sign, the type to
// which the data argument will be converted.
//
//	fetcher::%int::hi = uint8
//	fetcher::%int::low = uint8
//	fetcher::%int::top = uint8
//	fetcher::%int::bottom = uint8
//	frac::%int::hi = uint8
//	frac::%int::low = uint8
//	frac::%int::increment = uint8
//	frac::%int::count = uint8
//	music::%int::waveform = uint8
//	music::%int::freq = uint8
//	music::%int::count = uint8
//	rng = uint8
//	fastfetch = bool
//
// note that PutRegister() will panic() if the register or data string is invalid.
func (cart *dpcPlus) PutRegister(register string, data string) {
	// most data is expected to an integer (a uint8 specifically) so we try
	// to convert it here. if it doesn't convert then it doesn't matter
	d, _ := strconv.ParseUint(data, 16, 8)

	r := strings.Split(register, "::")
	switch r[0] {
	case "fetcher":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.Fetcher) {
			panic(fmt.Sprintf("unrecognised fetcher [%s]", register))
		}
		switch r[2] {
		case "hi":
			cart.state.registers.Fetcher[f].Hi = uint8(d)
		case "low":
			cart.state.registers.Fetcher[f].Low = uint8(d)
		case "top":
			cart.state.registers.Fetcher[f].Top = uint8(d)
		case "bottom":
			cart.state.registers.Fetcher[f].Bottom = uint8(d)
		default:
			panic(fmt.Sprintf("unrecognised variable [%s]", register))
		}
	case "frac":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.FracFetcher) {
			panic(fmt.Sprintf("unrecognised fetcher [%s]", register))
		}
		switch r[2] {
		case "hi":
			cart.state.registers.FracFetcher[f].Hi = uint8(d)
		case "low":
			cart.state.registers.FracFetcher[f].Low = uint8(d)
		case "increment":
			cart.state.registers.FracFetcher[f].Increment = uint8(d)
		case "count":
			cart.state.registers.FracFetcher[f].Count = uint8(d)
		default:
			panic(fmt.Sprintf("unrecognised variable [%s]", register))
		}
	case "music":
		f, err := strconv.Atoi(r[1])
		if err != nil || f > len(cart.state.registers.MusicFetcher) {
			panic(fmt.Sprintf("unrecognised fetcher [%s]", register))
		}
		switch r[2] {
		case "waveform":
			cart.state.registers.MusicFetcher[f].Waveform = uint32(d)
		case "freq":
			cart.state.registers.MusicFetcher[f].Freq = uint32(d)
		case "increment":
			cart.state.registers.MusicFetcher[f].Count = uint32(d)
		default:
			panic(fmt.Sprintf("unrecognised variable [%s]", register))
		}
	case "rng":
		cart.state.registers.RNG.Value = uint32(d)
	case "fastfetch":
		switch data {
		case "true":
			cart.state.registers.FastFetch = true
		case "false":
			cart.state.registers.FastFetch = false
		default:
			panic(fmt.Sprintf("unrecognised boolean state [%s]", data))
		}
	default:
		panic(fmt.Sprintf("unrecognised variable [%s]", register))
	}
}
