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
)

// CapperWriter is an implementation of io.Writer that stops buffering once a
// predefined size is reached.
type CappedWriter struct {
	buffer []byte
	size   int
}

// NewCappedWriter is the preferred method of initialisation for the
// CappedWriter type.
func NewCappedWriter(size int) (*CappedWriter, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size for CappedWriter (%d)", size)
	}
	return &CappedWriter{
		size:   size,
		buffer: make([]byte, 0, size),
	}, nil
}

func (r *CappedWriter) String() string {
	return string(r.buffer)
}

// Reset empties the ring writer's buffer
func (r *CappedWriter) Reset() {
	r.buffer = r.buffer[:0]
}

// Write implements io.Writer
func (r *CappedWriter) Write(p []byte) (n int, err error) {
	remaining := r.size - len(r.buffer)

	// no space remaining
	if remaining == 0 {
		return 0, nil
	}

	// plenty of space remaining
	if len(p) < remaining {
		r.buffer = append(r.buffer, p...)
		return len(p), nil
	}

	// limit how much we append
	r.buffer = append(r.buffer, p[:remaining]...)
	return remaining, nil
}
