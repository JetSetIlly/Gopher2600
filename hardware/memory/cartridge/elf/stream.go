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

package elf

import "fmt"

type streamEntry struct {
	addr uint16
	data uint8
}

func (s streamEntry) String() string {
	return fmt.Sprintf("%04x=%02x", s.addr, s.data)
}

type stream struct {
	active bool
	stream [1000]streamEntry
	ptr    int

	drain    bool
	drainPtr int
	drainTop int
}

func (s *stream) startDrain() {
	s.drain = true
	s.drainTop = s.ptr
	s.drainPtr = 0
}

func (s *stream) push(e streamEntry) {
	s.stream[s.ptr] = e
	s.ptr++

	// the stream can be pushed to even if the drain has started. this can
	// happen when the pushBoundary has been reached but there are still bytes
	// to be pushed from the current strongarm function
	if s.drain {
		s.drainTop = s.ptr
	} else {
		// the pushBoundary prevents out-of-bounds errors in the event of an
		// instruction pushing more bytes than are available. a sufficiently
		// high boundary value means that the drain will start before the next
		// strongarm function is reached but allowing the current function to
		// complete
		const pushBoundary = 10
		if s.ptr >= len(s.stream)-pushBoundary {
			s.startDrain()
		}
	}
}

func (s *stream) pull() streamEntry {
	e := s.stream[s.drainPtr]
	s.drainPtr++
	if s.drainPtr >= s.drainTop {
		s.drain = false
		s.ptr = 0
	}
	return e
}

func (s *stream) peekAddr() uint16 {
	return s.stream[s.drainPtr].addr
}
