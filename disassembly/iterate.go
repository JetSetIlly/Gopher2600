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

	"github.com/jetsetilly/gopher2600/errors"
)

// Iterate faciliates traversal of the disassembly
type Iterate struct {
	dsm       *Disassembly
	typ       EntryType
	bank      int
	idx       int
	lastEntry *Entry
}

// NewIteration initialises a new iteration of a dissasembly bank
func (dsm *Disassembly) NewIteration(typ EntryType, bank int) (*Iterate, error) {
	if bank > len(dsm.Entries) {
		return nil, errors.New(errors.IterationError, fmt.Sprintf("no bank %d in disassembly", bank))
	}

	itr := &Iterate{
		dsm:  dsm,
		typ:  typ,
		bank: bank,
	}

	return itr, nil
}

// Start new iteration from the first instance of the EntryType specified in
// NewIteration.
func (itr *Iterate) Start() *Entry {
	itr.idx = 0
	return itr.Next()
}

// Next entry in the disassembly of the previously specified type. Returns nil
// if end of disassembly has been reached.
func (itr *Iterate) Next() *Entry {
	var e *Entry

	for itr.idx < len(itr.dsm.Entries[itr.bank]) {
		e = itr.dsm.Entries[itr.bank][itr.idx]
		if e != nil && e.Type >= itr.typ {
			itr.idx++
			break // for loop
		}
		itr.idx++
	}

	itr.lastEntry = e

	return e
}

// SkipNext n entries and return that Entry. An n value of < 0 returns the most
// recent value in the iteration
func (itr *Iterate) SkipNext(n int) *Entry {
	e := itr.lastEntry

	for n > 0 {
		e = itr.Next()
		n--
	}

	return e
}
