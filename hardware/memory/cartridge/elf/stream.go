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

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/logger"
)

type streamEntry struct {
	addr     uint16
	data     uint8
	busstuff bool
}

func (s streamEntry) String() string {
	return fmt.Sprintf("%04x=%02x (stuff=%v)", s.addr, s.data, s.busstuff)
}

// the pushBoundary prevents out-of-bounds errors in the event of a strongarm
// instruction pushing more bytes than are available. a sufficiently high
// boundary value means that next function to execute will complete without
// exceeding the bounds of the array
//
// the high value is to accomodate the relatively high byte count of the
// vcsCopyOverblankToRiotRam() function, which consumes about 200 bytes. a
// typical function will require no more than half-a-dozen bytes but the copy
// function represents a significant block of 6507 code
const pushBoundary = 200

type stream struct {
	// diabled indicates that the stream is not available and should not be
	// activated
	disabled bool

	// whether the stream is acutally active. setting this value should only be
	// true if the disabled field is false.
	// ie. only 'active = !disabled' or 'active = false'
	active bool

	stream [1000 + pushBoundary]streamEntry
	ptr    int

	drain    bool
	drainPtr int
	drainTop int

	// indicates that a data bus snooping needs resolving
	snoopDataBus bool
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
		// see comment about the pushBoundary
		if s.ptr > len(s.stream)-pushBoundary {
			s.startDrain()
		}
	}
}

func (s *stream) pull() streamEntry {
	if !s.drain {
		logger.Log(logger.Allow, "ELF", "unexpected call to stream.pull(). returning zero data")
		return streamEntry{}
	}
	e := s.stream[s.drainPtr]
	s.drainPtr++
	if s.drainPtr >= s.drainTop {
		s.drain = false
		s.ptr = 0
	}
	return e
}

func (s *stream) peek() streamEntry {
	if s.drain {
		return s.stream[s.drainPtr]
	}
	return streamEntry{}
}
