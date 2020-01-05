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

package polycounter

import (
	"fmt"
	"gopher2600/errors"
)

// Polycounter counts from 0 to Limit. can be used to index a polycounter
// table
type Polycounter struct {
	numBits int
	table   []string
	count   int
	max     int
}

// New is the preferred method of initialisation for the Polycounter type
func New(numBits int) (*Polycounter, error) {
	pcnt := &Polycounter{
		numBits: numBits,
		max:     (1 << numBits) - 1,
	}

	var p int

	mask := pcnt.max
	shift := numBits - 1
	format := fmt.Sprintf("%%0%db", numBits)

	pcnt.table = make([]string, 1<<numBits)
	pcnt.table[0] = fmt.Sprintf(format, 0)

	for i := 1; i < len(pcnt.table); i++ {
		p = ((p & (mask - 1)) >> 1) | (((p&1)^((p>>1)&1))^mask)<<shift
		p = p & mask
		pcnt.table[i] = fmt.Sprintf(format, p)
	}

	// sanity check that the table has looped correctly
	if pcnt.table[len(pcnt.table)-1] != pcnt.table[0] {
		return nil, errors.New(errors.PolycounterError, fmt.Sprintf("error creating %d bit polycounter", numBits))
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	pcnt.table[len(pcnt.table)-1] = fmt.Sprintf(format, pcnt.max)

	return pcnt, nil
}

func (pcnt Polycounter) String() string {
	// assumes maximum limit of 2 digits
	return fmt.Sprintf("%s (%02d)", pcnt.ToBinary(), pcnt.count)
}

// Reset is a convenience function to reset count value to 0
func (pcnt *Polycounter) Reset() {
	pcnt.count = 0
}

// Tick advances the Polycounter and resets when it reaches the limit.
// returns true if counter has reset
func (pcnt *Polycounter) Tick() bool {
	pcnt.count++
	if pcnt.count >= pcnt.max {
		pcnt.count = 0
		return true
	}

	return false
}

// Count reports the current polycounter value as an integer For the bit
// pattern representation, use the ToBinary() function.
func (pcnt *Polycounter) Count() int {
	return pcnt.count
}

// ToBinary returns the bit pattern of the current polycounter value
func (pcnt *Polycounter) ToBinary() string {
	return pcnt.table[pcnt.count]
}

// Match returns the index of the specified pattern
func (pcnt *Polycounter) Match(pattern string) (int, error) {
	for i := 0; i < len(pcnt.table); i++ {
		if pcnt.table[i] == pattern {
			return i, nil
		}
	}
	return 0, errors.New(errors.PolycounterError, fmt.Sprintf("could not find pattern (%s) in %d bit lookup table", pattern, pcnt.numBits))
}
