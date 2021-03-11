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
	"fmt"
	"io"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
)

// IterateBanks faciliates traversal over all the banks in a cartridge.
type IterateBanks struct {
	dsm  *Disassembly
	bank int

	// number of banks in cart iteration
	BankCount int
}

// NewBanksIteration is the preferred method of initialisation for the IterateCart type.
func (dsm *Disassembly) NewBanksIteration() *IterateBanks {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	citr := &IterateBanks{
		BankCount: len(dsm.entries),
		dsm:       dsm,
	}

	return citr
}

// Start new iteration from the first bank.
func (citr *IterateBanks) Start() (int, bool) {
	citr.dsm.crit.Lock()
	defer citr.dsm.crit.Unlock()
	citr.bank = -1
	return citr.next()
}

// The next bank in the cartidge. Returns (-1, false) if there are no more banks.
func (citr *IterateBanks) Next() (int, bool) {
	citr.dsm.crit.Lock()
	defer citr.dsm.crit.Unlock()
	return citr.next()
}

func (citr *IterateBanks) next() (int, bool) {
	if citr.bank+1 >= citr.BankCount {
		return -1, false
	}
	citr.bank++
	return citr.bank, true
}

// IterateEntries iterates over all entries in a cartridge bank.
//
// Instances of Entry returned by Start(), Next() and SkipNext() are copies of
// the disassembly entry, so the Iterate mechanism is suitable for use in a
// goroutine different to that which is handling (eg. updating) the disassembly
// itslef.
type IterateEntries struct {
	dsm *Disassembly

	// the bank we're iterating over
	bank int

	// include entry in iteration if it meets at least this EntryLevel
	minLevel EntryLevel

	// address the iteration is focused on. this entry will be included event
	// if the EntryLevel does not meet the minimum level
	FocusAddr []uint16

	// the iteration count at which the first entry in FocusAddr will be found
	FocusAddrCt int

	// total number of entries in iteration with the specified minimum level
	EntryCount int

	// the number of those entries with a label
	LabelCount int

	idx       int
	lastEntry *Entry
}

// NewEntriesIteration initialises a new iteration of a dissasembly bank. The minLevel
// argument specifies the minimum entry level which should be returned in the
// iteration. So, using the following as a guide:
//
//	dead < decoded < blessed
//
// Specifying a minLevel of EntryLevelDecode will iterate *only* entries of
// EntryLevelDecode. A minLevel of EntryLevelNaive on the other hand, will
// iterate through entries of EntryLevelNaive *and* EntryLevelDecode. A
// minLevel of EntryLevelDead will iterate through *all* Entries.
//
// The final argument, focusAddr, specifies the addresses that must be included
// in the iteration regardless of EntryLevel. Can be left empty.
func (dsm *Disassembly) NewEntriesIteration(minLevel EntryLevel, bank int, focusAddr ...uint16) (*IterateEntries, error) {
	dsm.crit.Lock()
	defer dsm.crit.Unlock()

	// silently reject iterations for non-existent banks. this may happen more
	// often than you think. for example, loading a new cartridge with fewer
	// banks than the current cartridge at the exact moment an illegal bank is
	// being drawn by the sdlimgui disassembly window.
	if bank >= len(dsm.entries) {
		return nil, curated.Errorf("no bank %d in disasm", bank)
	}

	eitr := &IterateEntries{
		dsm:         dsm,
		bank:        bank,
		minLevel:    minLevel,
		FocusAddr:   make([]uint16, len(focusAddr)),
		FocusAddrCt: -1,
	}

	// normalise addresses
	for i := range focusAddr {
		eitr.FocusAddr[i] = (focusAddr[i] & memorymap.CartridgeBits) | memorymap.OriginCart
	}

	// count entries
	for _, e := range dsm.entries[bank] {
		if e == nil {
			return nil, curated.Errorf("disassembly not complete")
		}

		// where in the iteration we'll encounter the FocusAddr entry
		if len(eitr.FocusAddr) > 0 && eitr.FocusAddr[0] == e.Result.Address&memorymap.CartridgeBits|memorymap.OriginCart {
			eitr.FocusAddrCt = eitr.EntryCount + eitr.LabelCount
		}

		// count the number of entries of the minimum level
		if e.Level >= minLevel {
			eitr.EntryCount++

			// count entries (of the minimum level) with a label
			if e.Label.String() != "" {
				eitr.LabelCount++
			}
		} else {
			for _, f := range eitr.FocusAddr {
				if f == e.Result.Address&memorymap.CartridgeBits|memorymap.OriginCart {
					// include address in the count even though it doesn meet minimum level
					eitr.EntryCount++
				}
			}
		}
	}

	return eitr, nil
}

// Start new iteration from the first instance of the EntryLevel specified in
// NewEntriesIteration.
func (eitr *IterateEntries) Start() (int, *Entry) {
	eitr.idx = -1
	return eitr.next()
}

// Next entry in the disassembly of the previously specified type.
//
// Returns (-1, nil) if end of disassembly has been reached.
func (eitr *IterateEntries) Next() (int, *Entry) {
	return eitr.next()
}

// SkipNext n entries and return that Entry. An n value of < 0 returns the most
// recent value in the iteration
//
// The skipLabels argument indicates that an entry with a label should count as
// two entries. This is useful for the sdlimgui disassembly window's list
// clipper (and maybe nothing else).
//
// Returns (-1, nil) if end of disassembly has been reached.
func (eitr *IterateEntries) SkipNext(n int, skipLabels bool) (int, *Entry) {
	e := eitr.lastEntry

	for n > 0 {
		if e == nil {
			return -1, nil
		}

		n--

		if e.Label.String() != "" {
			n--
		}

		_, e = eitr.next()
	}
	return eitr.idx, e
}

func (eitr *IterateEntries) next() (int, *Entry) {
	eitr.dsm.crit.Lock()
	defer eitr.dsm.crit.Unlock()

	if eitr.idx+1 >= len(eitr.dsm.entries[eitr.bank]) {
		return -1, nil
	}

	// find next entry to return
	eitr.idx++
	done := false
	for !done {
		a := eitr.dsm.entries[eitr.bank][eitr.idx]
		if a.Level >= eitr.minLevel {
			break // for idx loop
		}

		// entry doesn't match minimum level but focusAddr might
		for _, f := range eitr.FocusAddr {
			if f == a.Result.Address&memorymap.CartridgeBits|memorymap.OriginCart {
				done = true
				break // for idx loop
			}
		}

		eitr.idx++
		done = done || eitr.idx >= len(eitr.dsm.entries[eitr.bank])
	}

	// for loop went past the end of the iterable content so return nothing
	if eitr.idx >= len(eitr.dsm.entries[eitr.bank]) {
		return -1, nil
	}

	eitr.lastEntry = eitr.dsm.entries[eitr.bank][eitr.idx]

	return eitr.idx, makeCopyofEntry(*eitr.lastEntry)
}

// we don't want to return the actual entry in the disassembly because it will
// result in a race condition erorr if the entry is updated at the same time as
// we're dealing with the iteration.
func makeCopyofEntry(e Entry) *Entry {
	return &e
}

// IterateBlessed visits every entry in the disassembly optionally writing to
// output. This will very often be more convenient that looping over
// IterateBanks and IterateEntries.
//
// Entries can be filtered with function f(). Strings returned from the filter
// function will be trimmed of trailing newline characters. Strings of length
// greater than zero thereafter, will be written to output.
//
// The string returned by the filter function can be multiline if required.
//
// If no output is required and all the necessary work is done in the filter
// function then an io.Writer value of nil is acceptable.
//
// If the io.Writer value is not nil then a sequential list of entries returned
// by the filter function. The sequnce of entries will be in bank and address
// order.
//
// Banks will be labelled (with a header) if output has been written for that
// bank. For example:
//
// --- Bank 2 ---
//  entry
//  entry
//
// --- Bank 5 ---
//  entry
//
// In this example there are no filtered entries for banks 0, 1, 3 or 4. There
// are entries for banks 2 and 5 so those entries have a header indicating the
// bank.
func (dsm *Disassembly) IterateBlessed(output io.Writer, f func(*Entry) string) error {
	hasOutput := false

	// look at every bank in the disassembly
	citr := dsm.NewBanksIteration()
	citr.Start()
	for b, ok := citr.Start(); ok; b, ok = citr.Next() {
		// create a new iteration for the bank
		eitr, err := dsm.NewEntriesIteration(EntryLevelBlessed, b)
		if err != nil {
			return err
		}

		bankHeader := false

		// iterate through disassembled bank
		for _, e := eitr.Start(); e != nil; _, e = eitr.Next() {
			s := f(e)
			s = strings.TrimSuffix(s, "\n")

			if output != nil && len(s) > 0 {
				// if we've not yet printed head for the current bank then print it now
				if !bankHeader {
					if hasOutput {
						output.Write([]byte("\n"))
					}
					output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", b)))
					bankHeader = true
					hasOutput = true
				}
				output.Write([]byte(s))
				output.Write([]byte("\n"))
			}
		}
	}

	return nil
}
