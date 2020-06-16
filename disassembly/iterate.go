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

// Iterate faciliates traversal of the disassembly.
//
// Instances of Entry returned by Start(), Next() and SkipNext() are copies of
// the disassembly entry, so the Iterate mechanism is suitable for use in a
// goroutine different to that which is handling (eg. updating) the disassembly
// itslef.
type Iterate struct {
	dsm       *Disassembly
	minLevel  EntryLevel
	bank      int
	idx       int
	lastEntry *Entry
}

// NewIteration initialises a new iteration of a dissasembly bank. The minLevel
// argument specifies the minumum entry level which should be returned in the
// iteration. So, using the following as a guide:
//
//	dead < decoded < blessed
//
// Specifying a minLevel of EntryLevelDecode will iterate *only* entries of
// EntryLevelDecode. A minLevel of EntryLevelNaive on the other hand, will
// iterate through entries of EntryLevelNaive *and* EntryLevelDecode. A
// minLevel of EntryLevelDead will iterate through *all* Entries.
//
// The function returns an instance of Iterate, a count of the number of
// entries the correspond to the minLevel (see above), and any error.
func (dsm *Disassembly) NewIteration(minLevel EntryLevel, bank int) (*Iterate, int, error) {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// silently reject iterations for non-existent banks. this may happen more
	// often than you think. for example, loading a new cartridge with fewer
	// banks than the current cartridge at the exact moment an illegal bank is
	// being drawn by the sdlimgui disassembly window.
	if bank >= len(dsm.reference) || bank >= len(dsm.counts) {
		return nil, 0, errors.New(errors.IterationError, fmt.Sprintf("no bank %d in disassembly", bank))
	}

	itr := &Iterate{
		dsm:      dsm,
		minLevel: minLevel,
		bank:     bank,
	}

	count := 0

	switch minLevel {
	case EntryLevelDead:
		count = dsm.counts[bank][EntryLevelDead]
		count += dsm.counts[bank][EntryLevelDecoded]
		count += dsm.counts[bank][EntryLevelBlessed]
		count += dsm.counts[bank][EntryLevelExecuted]

	case EntryLevelDecoded:
		count = dsm.counts[bank][EntryLevelDecoded]
		count += dsm.counts[bank][EntryLevelBlessed]
		count += dsm.counts[bank][EntryLevelExecuted]

	case EntryLevelBlessed:
		count = dsm.counts[bank][EntryLevelBlessed]
		count += dsm.counts[bank][EntryLevelExecuted]

	case EntryLevelExecuted:
		count = dsm.counts[bank][EntryLevelExecuted]
	}

	return itr, count, nil
}

// Start new iteration from the first instance of the EntryLevel specified in
// NewIteration.
func (itr *Iterate) Start() *Entry {
	itr.idx = 0
	return itr.Next()
}

// Next entry in the disassembly of the previously specified type. Returns nil
// if end of disassembly has been reached.
func (itr *Iterate) Next() *Entry {
	itr.dsm.crit.Lock()
	defer itr.dsm.crit.Unlock()

	if itr.idx >= len(itr.dsm.reference[itr.bank]) {
		return nil
	}

	itr.idx++

	for itr.idx < len(itr.dsm.reference[itr.bank]) && itr.dsm.reference[itr.bank][itr.idx].Level < itr.minLevel {
		itr.idx++
	}

	if itr.idx >= len(itr.dsm.reference[itr.bank]) {
		return nil
	}

	itr.lastEntry = itr.dsm.reference[itr.bank][itr.idx]

	return makeCopyofEntry(*itr.lastEntry)
}

// we don't want to return the actual entry in the disassembly because it will
// result in a race condition erorr if the entry is updated at the same time as
// we're dealing with the iteration.
func makeCopyofEntry(e Entry) *Entry {
	return &e
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
