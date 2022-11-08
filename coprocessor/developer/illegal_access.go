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

package developer

import "fmt"

// IllegalAccessEntry is a single entry in the illegal access log.
type IllegalAccessEntry struct {
	Event      string
	PC         uint32
	AccessAddr uint32

	// number of times this specific illegal access has been seen
	Count int

	// the source line of the PC address. field can be nil
	SrcLine *SourceLine

	// whether access address was reported as being a "null access". when this
	// is true the illegal access is very likely because of a null pointer
	// dereference
	IsNullAccess bool
}

// IllegalAccess records memory accesses by the coprocesser that are "illegal".
type IllegalAccess struct {
	// entries are keyed by concatanation of PC and AccessAddr expressed as a
	// 16 character string
	entries map[string]*IllegalAccessEntry

	// all the accesses in order of the first time they appear. the Count field
	// in the IllegalAccessEntry can be used to see if that entry was seen more
	// than once *after* the first appearance
	Log []*IllegalAccessEntry

	// is true once a stack collision has been detected. once a stack collision
	// has occured then subsequent illegal accesses cannot be trusted and will
	// likely not be logged
	HasStackCollision bool
}

// BorrowIllegalAccess will lock the illegal access log for the duration of the
// supplied fucntion, which will be executed with the illegal access log as an
// argument.
func (dev *Developer) BorrowIllegalAccess(f func(*IllegalAccess)) {
	dev.illegalAccessLock.Lock()
	defer dev.illegalAccessLock.Unlock()
	f(&dev.illegalAccess)
}

// IllegalAccess implements the CartCoProcDeveloper interface.
func (dev *Developer) NullAccess(event string, pc uint32, addr uint32) string {
	return dev.logAccess(event, pc, addr, true)
}

// IllegalAccess implements the CartCoProcDeveloper interface.
func (dev *Developer) IllegalAccess(event string, pc uint32, addr uint32) string {
	return dev.logAccess(event, pc, addr, false)
}

// IllegalAccess implements the CartCoProcDeveloper interface.
func (dev *Developer) StackCollision(pc uint32, addr uint32) string {
	dev.illegalAccess.HasStackCollision = true
	return dev.logAccess("Stack Collision", pc, addr, false)
}

// logAccess adds an illegal or null access event to the log. includes source code lookup
func (dev *Developer) logAccess(event string, pc uint32, addr uint32, isNullAccess bool) string {
	dev.sourceLock.Lock()
	defer dev.sourceLock.Unlock()

	// get/create illegal access entry
	accessKey := fmt.Sprintf("%08x%08x", addr, pc)
	e, ok := dev.illegalAccess.entries[accessKey]
	if ok {
		// we seen the illegal access before - increase count
		e.Count++
	} else {
		e = &IllegalAccessEntry{
			Event:        event,
			PC:           pc,
			AccessAddr:   addr,
			Count:        1,
			IsNullAccess: isNullAccess,
		}

		dev.illegalAccess.entries[accessKey] = e

		// if we have source code available we assign the source line to the
		// illegal access entry. it's possible for the source line to be a
		// "stub" source line
		if dev.source != nil {
			e.SrcLine = dev.source.linesByAddress[uint64(pc)]

			// it is sometimes possible to have source (ie dev.source != nil)
			// but for there to be no actual source files. in these instances
			// we need to create the a stub entry for the line as we go along
			if e.SrcLine == nil {
				e.SrcLine = createStubLine(nil)
			}

			// inidcate that the source line has been responsble for an illegal access
			e.SrcLine.Bug = true
		} else {
			e.SrcLine = createStubLine(nil)
		}

		// if we do not have source code available then the entry will have a
		// nil SrcLine field. this is fine

		// record entry
		dev.illegalAccess.entries[accessKey] = e

		// update log
		dev.illegalAccess.Log = append(dev.illegalAccess.Log, e)
	}

	// no source line information so return empty line
	if e.SrcLine == nil || e.SrcLine.IsStub() {
		return ""
	}

	// return formatted information about the illegal access using the source line for information
	return fmt.Sprintf("%s %s\n%s", e.SrcLine.String(), e.SrcLine.Function.Name, e.SrcLine.PlainContent)
}
