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

package polycounter

import (
	"fmt"
)

var table6bit []string

// initialise table for a 6bit polycounter. the VCS only uses 6 bit
// polycounters but the following method can be used to produce tables of any
// length.
func init() {
	mask := (1 << 6) - 1
	format := fmt.Sprintf("%%0%db", 6)

	table6bit = make([]string, 1<<6)
	table6bit[0] = fmt.Sprintf(format, 0)

	var p int

	for i := 1; i < len(table6bit); i++ {
		p = ((p & (mask - 1)) >> 1) | (((p&1)^((p>>1)&1))^mask)<<5
		p &= mask
		table6bit[i] = fmt.Sprintf(format, p)
	}

	// sanity check that the table has looped correctly
	if table6bit[len(table6bit)-1] != table6bit[0] {
		panic("error creating 6 bit polycounter")
	}

	// force the final value to be the invalid polycounter value. this is only
	// ever useful for specifying the reset pattern
	table6bit[len(table6bit)-1] = fmt.Sprintf(format, mask)
}
