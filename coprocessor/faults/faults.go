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

package faults

import (
	"fmt"
	"io"
)

// Category classifies the approximate reason for a memory fault
type Category string

// List of valid Category values
const (
	NullDereference  Category = "null dereference"
	MisalignedAccess Category = "misaligned access"
	StackCollision   Category = "stack collision"
	IllegalAddress   Category = "illegal address"
	UndefinedSymbol  Category = "undefined symbol"
	ProgramMemory    Category = "program memory"
)

// Entry is a single entry in the fault log
type Entry struct {
	Category Category

	// description of the event that triggered the memory fault
	Event string

	// addresses related to the fault
	InstructionAddr uint32
	AccessAddr      uint32

	// number of times this specific illegal access has been seen
	Count int
}

func (e Entry) String() string {
	return fmt.Sprintf("%s: %s: %08x (PC: %08x)", e.Category, e.Event, e.AccessAddr, e.InstructionAddr)
}

// Faults records memory accesses by the coprocesser that are "illegal".
type Faults struct {
	// entries are keyed by concatanation of InstructionAddr and AccessAddr expressed as a
	// 16 character string
	entries map[string]*Entry

	// all the accesses in order of the first time they appear. the Count field
	// in the IllegalAccessEntry can be used to see if that entry was seen more
	// than once *after* the first appearance
	Log []*Entry

	// is true once a stack collision has been detected. once a stack collision
	// has occured then subsequent illegal accesses cannot be trusted and will
	// likely not be logged
	HasStackCollision bool
}

func NewFaults() Faults {
	return Faults{
		entries: make(map[string]*Entry),
	}
}

// Clear all entries from faults log. Does not clear the HasStackCollision flag
func (flt *Faults) Clear() {
	clear(flt.entries)
	flt.Log = flt.Log[:0]
}

// WriteLog writes the list of faults in the order they were added
func (flt Faults) WriteLog(w io.Writer) {
	for _, e := range flt.Log {
		w.Write([]byte(e.String()))
	}
}

// NewEntry adds a new entry to the list of faults
func (flt *Faults) NewEntry(event string, category Category, instructionAddr uint32, accessAddr uint32) {
	key := fmt.Sprintf("%08x%08x", instructionAddr, accessAddr)

	e, found := flt.entries[key]
	if !found {
		e = &Entry{
			Category:        category,
			Event:           event,
			InstructionAddr: instructionAddr,
			AccessAddr:      accessAddr,
		}

		// record entry
		flt.entries[key] = e

		// update log
		flt.Log = append(flt.Log, e)
	}

	// increase the count for this entry
	e.Count++

	// if this is a stack collision then record that fact
	if category == StackCollision {
		flt.HasStackCollision = true
	}
}
