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

package crunched

type quick struct {
	crunched       bool
	data           []byte
	uncrunchedSize int
}

// NewQuick returns an implementation of the Data interface that is intended to
// perform quickly on both crunching and decrunching.
//
// For simplicity, the minimum amount of data allocated will be 4 bytes.
func NewQuick(size int) Data {
	size = max(size, 4)
	return &quick{
		data:           make([]byte, size),
		uncrunchedSize: size,
	}
}

// IsCrunched returns true if data is currently crunched
//
// This function implements the Data interface
func (c *quick) IsCrunched() bool {
	return c.crunched
}

// Size returns the uncrunched size and the current size of the data. If the
// data is currently crunched then the two values will be the same
//
// This function implements the Data interface
func (c *quick) Size() (int, int) {
	return c.uncrunchedSize, len(c.data)
}

// Data returns a pointer to the uncrunched data
//
// This function implements the Data interface
func (c *quick) Data() *[]byte {
	if c.crunched {
		// sanity check. with the current RLE method the number of bytes in the
		// crunched data should be a multiple of two
		if len(c.data)&0x01 == 0x01 {
			panic("crunched data should have an even number of bytes")
		}

		// make a reference to the crunched data before creating space for the
		// uncrunched data
		working := c.data
		c.data = make([]byte, c.uncrunchedSize)

		// undo the RLE process
		var idx int
		for i := 0; i < len(working); i += 2 {
			for r := 0; r <= int(working[i+1]); r++ {
				c.data[idx] = working[i]
				idx++
			}
		}

		// data is now uncrunched
		c.crunched = false
	}

	return &c.data
}

// Snapshot makes a copy of the data and crunching it if required. The data will
// be uncrunched automatically when Data() function is called
//
// This function implements the Data interface
func (c *quick) Snapshot() Data {
	d := *c

	if !d.crunched {
		working := make([]byte, d.uncrunchedSize)

		var ct int
		var idx int
		working[idx] = c.data[0]

		// assume crunching has succeeded unless explicitely told otherwise
		d.crunched = true

		// very basic RLE algorithm:
		// 1) each byte is followed by a count value
		// 2) maximum count value is 255
		for _, v := range c.data[1:] {
			if v == working[idx] && ct < 255 {
				ct++
			} else {
				// check that the crunched data isn't getting too large. we'll
				// be adding two bytes to the crunch stream so the check here is
				// to make sure that won't overflow the size of the array
				if idx >= len(working)-2 {
					d.crunched = false
					break // for loop
				}

				// output count to the crunch stream
				idx++
				working[idx] = byte(ct)

				// output new byte to crunch stream
				idx++
				working[idx] = v

				// count will begin again with the new byte
				ct = 0
			}
		}

		// if the data has been crunched then allocate just enough memory to
		// store the crunched data before returning
		if d.crunched {
			idx++
			working[idx] = byte(ct)
			d.data = make([]byte, idx+1)
			copy(d.data, working[:idx+1])
			return &d
		}

		// if data is not crunched then we intentionally fall through to the
		// plain data copy below
	}

	// copy data as it exists now. this may be crunched data or uncrunched data.
	// it doesn't matter either way
	d.data = make([]byte, len(c.data))
	copy(d.data, c.data)

	return &d
}

// Inspect returns data in the current state. In other words, the data will
// not be decrunched as it would be with the Data() function
//
// This function implements the Peep interface
func (c *quick) Inspect() *[]byte {
	return &c.data
}
