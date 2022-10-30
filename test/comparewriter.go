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

// CompareWriter is an implementation of the io.CompareWriter interface. It should be used to
// capture output and to compare with predefined strings.
type CompareWriter struct {
	buffer []byte
}

func (tw *CompareWriter) Write(p []byte) (n int, err error) {
	tw.buffer = append(tw.buffer, p...)
	return len(p), nil
}

// clear string empties the buffer.
func (tw *CompareWriter) Clear() {
	tw.buffer = tw.buffer[:0]
}

// Compare buffered output with predefined/example string.
func (tw *CompareWriter) Compare(s string) bool {
	return s == string(tw.buffer)
}

// implements Stringer interface.
func (tw *CompareWriter) String() string {
	return string(tw.buffer)
}
