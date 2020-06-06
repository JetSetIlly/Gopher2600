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
	"strings"
)

// DPCplusRegisters implements the bus.CartRegisters interface
type DPCplusRegisters struct {
	Fetcher      [8]dataFetcher
	FracFetcher  [8]fractionalDataFetcher
	MusicFetcher [3]musicDataFetcher

	// random number generator
	RNG randomNumberFetcher

	// fast fetch read mode
	FastFetch bool
}

// String implements the bus.CartDebugBus interface
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
	if df.Low == 0x00 {
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
