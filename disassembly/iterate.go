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

package disassembly

import (
	"github.com/jetsetilly/gopher2600/curated"
)

// IterateCart faciliates traversal over all the banks in a cartridge.
type IterateCart struct {
	dsm  *Disassembly
	bank int

	// number of banks in cart iteration
	BankCount int
}

// NewCartIteration is the preferred method of initialisation for the
// IterateCart type.
func (dsm *Disassembly) NewCartIteration() *IterateCart {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	citr := &IterateCart{
		BankCount: len(dsm.entries),
		dsm:       dsm,
	}

	return citr
}

// Start new iteration from the first bank.
func (citr *IterateCart) Start() (int, bool) {
	citr.dsm.crit.Lock()
	defer citr.dsm.crit.Unlock()
	citr.bank = -1
	return citr.next()
}

// The next bank in the cartidge. Returns (-1, false) if there are no more banks.
func (citr *IterateCart) Next() (int, bool) {
	citr.dsm.crit.Lock()
	defer citr.dsm.crit.Unlock()
	return citr.next()
}

func (citr *IterateCart) next() (int, bool) {
	if citr.bank+1 >= citr.BankCount {
		return -1, false
	}
	citr.bank++
	return citr.bank, true
}

// IterateBank faciliates traversal a specific bank.
//
// Instances of Entry returned by Start(), Next() and SkipNext() are copies of
// the disassembly entry, so the Iterate mechanism is suitable for use in a
// goroutine different to that which is handling (eg. updating) the disassembly
// itslef.
type IterateBank struct {
	dsm       *Disassembly
	minLevel  EntryLevel
	bank      int
	idx       int
	lastEntry *Entry

	// total number of entries in iteration with the specified minimum level
	EntryCount int

	// the number of those entries with a label
	LabelCount int
}

// NewBankIteration initialises a new iteration of a dissasembly bank. The minLevel
// argument specifies the minimum entry level which should be returned in the
// iteration. So, using the following as a guide:
//
//	dead < decoded < blessed
//
// Specifying a minLevel of EntryLevelDecode will iterate *only* entries of
// EntryLevelDecode. A minLevel of EntryLevelNaive on the other hand, will
// iterate through entries of EntryLevelNaive *and* EntryLevelDecode. A
// minLevel of EntryLevelDead will iterate through *all* Entries.
func (dsm *Disassembly) NewBankIteration(minLevel EntryLevel, bank int) (*IterateBank, error) {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// silently reject iterations for non-existent banks. this may happen more
	// often than you think. for example, loading a new cartridge with fewer
	// banks than the current cartridge at the exact moment an illegal bank is
	// being drawn by the sdlimgui disassembly window.
	if bank >= len(dsm.entries) {
		return nil, curated.Errorf("no bank %d in disasm", bank)
	}

	bitr := &IterateBank{
		dsm:      dsm,
		minLevel: minLevel,
		bank:     bank,
	}

	// count entries
	for _, a := range dsm.entries[bank] {
		if a == nil {
			return nil, curated.Errorf("disassembly not complete")
		}

		// count the number of entries of the minimum level
		if a.Level >= minLevel {
			bitr.EntryCount++

			// count entries (of the minimum level) with a label
			if a.Label.String() != "" {
				bitr.LabelCount++
			}
		}
	}

	return bitr, nil
}

// Start new iteration from the first instance of the EntryLevel specified in NewBankIteration.
func (bitr *IterateBank) Start() (int, *Entry) {
	bitr.idx = -1
	return bitr.next()
}

// Next entry in the disassembly of the previously specified type. Returns nil if end of disassembly has been reached.
func (bitr *IterateBank) Next() (int, *Entry) {
	return bitr.next()
}

// SkipNext n entries and return that Entry. An n value of < 0 returns the most
// recent value in the iteration
//
// The skipLabels argument indicates that an entry with a label should count as
// two entries. This is useful for the sdlimgui disassembly window's list
// clipper (and maybe nothing else).
func (bitr *IterateBank) SkipNext(n int, skipLabels bool) (int, *Entry) {
	e := bitr.lastEntry
	for n > 0 {
		n--

		if e.Label.String() != "" {
			n--
		}

		_, e = bitr.next()
	}
	return bitr.idx, e
}

func (bitr *IterateBank) next() (int, *Entry) {
	bitr.dsm.crit.Lock()
	defer bitr.dsm.crit.Unlock()

	if bitr.idx+1 >= len(bitr.dsm.entries[bitr.bank]) {
		return -1, nil
	}

	bitr.idx++

	for bitr.idx < len(bitr.dsm.entries[bitr.bank]) && bitr.dsm.entries[bitr.bank][bitr.idx].Level < bitr.minLevel {
		bitr.idx++
	}

	if bitr.idx >= len(bitr.dsm.entries[bitr.bank]) {
		return -1, nil
	}

	bitr.lastEntry = bitr.dsm.entries[bitr.bank][bitr.idx]

	return bitr.idx, makeCopyofEntry(*bitr.lastEntry)
}

// we don't want to return the actual entry in the disassembly because it will
// result in a race condition erorr if the entry is updated at the same time as
// we're dealing with the iteration.
func makeCopyofEntry(e Entry) *Entry {
	return &e
}
