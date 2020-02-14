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

package disassembly

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory/memorymap"
)

// Iterate faciliates traversal of the disassembly
type Iterate struct {
	bank    int
	entries [memorymap.AddressMaskCart + 1]*Entry
	idx     int
}

// NewIteration initialises a new iteration of a dissasembly bank
func (dsm *Disassembly) NewIteration(bank int) (*Iterate, error) {
	if bank > len(dsm.Entries) {
		return nil, errors.New(errors.IterationError, fmt.Sprintf("no bank %d in disassembly", bank))
	}

	itr := &Iterate{
		bank:    bank,
		entries: dsm.Entries[bank],
		idx:     0,
	}

	return itr, nil
}

// Start new iteration from first index. Not strictly needed if called
// immediately after NewIteration()
func (itr *Iterate) Start() *Entry {
	itr.idx = 1
	return itr.entries[0]
}

// Next entry at least of EntryType in the disassembly. Returns nil if end of
// disassembly has been reached.
func (itr *Iterate) Next(typ EntryType) *Entry {
	var e *Entry

	for itr.idx < len(itr.entries) {
		e = itr.entries[itr.idx]
		if e != nil && e.Type >= typ {
			itr.idx++
			break // for loop
		}
		itr.idx++
	}

	return e
}
