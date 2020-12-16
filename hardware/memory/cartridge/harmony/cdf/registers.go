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

import "strings"

// Registers implements mappers.Registers
type Registers struct {
	MusicFetcher [3]musicDataFetcher
}

func (r *Registers) initialise() {
	for i := range r.MusicFetcher {
		r.MusicFetcher[i].Waveform = 0x1b
	}
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
