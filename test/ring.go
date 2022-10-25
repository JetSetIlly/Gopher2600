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

package test

import (
	"fmt"
	"strings"
)

type RingWriter struct {
	buffer  []byte
	size    int
	cursor  int
	wrapped bool
}

func NewRingWriter(size int) (*RingWriter, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size for RingWriter (%d)", size)
	}
	return &RingWriter{
		size:   size,
		buffer: make([]byte, size),
	}, nil
}

func (r *RingWriter) String() string {
	var s strings.Builder

	if r.wrapped {
		s.WriteString(string(r.buffer[r.cursor:]))
		s.WriteString(string(r.buffer[:r.cursor]))
	} else {
		s.WriteString(string(r.buffer[:r.cursor]))
	}

	return s.String()
}

func (r *RingWriter) Reset() {
	r.cursor = 0
	r.wrapped = false
}

// Write implements io.Writer
func (r *RingWriter) Write(p []byte) (n int, err error) {
	l := len(p)

	// new text is larger than ring so simply reset the ring and continue as
	// normal
	if l > r.size {
		r.cursor = 0
		r.wrapped = false
	}

	// copy p to buffer, accounting for any wrapping as required
	l = r.size - r.cursor
	copy(r.buffer[r.cursor:], p)
	if len(p) >= l {
		r.wrapped = true
		copy(r.buffer, p[l:])
	}

	// adjust cursor
	r.cursor = ((r.cursor + len(p)) % r.size)

	return len(p), nil
}
