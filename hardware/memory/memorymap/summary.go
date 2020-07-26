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

package memorymap

import (
	"fmt"
	"strings"
)

// Summary returns a single multiline string detailing all the areas in memory.
// Useful for reference.
func Summary() string {
	var area, current Area
	var a, sa uint16

	s := strings.Builder{}

	// look up area of first address in memory
	_, current = MapAddress(uint16(0), true)

	// for every address in the range 0 to MemtopCart...
	for a = uint16(1); a <= MemtopCart; a++ {
		// ...get the area name of that address.
		_, area = MapAddress(a, true)

		// if the area has changed print out the summary line...
		if area != current {
			s.WriteString(fmt.Sprintf("%04x -> %04x\t%s\n", sa, a-uint16(1), current.String()))

			// ...update current area and start address of the area
			current = area
			sa = a
		}
	}

	// write last line of summary
	s.WriteString(fmt.Sprintf("%04x -> %04x\t%s\n", sa, a-uint16(1), area.String()))

	return s.String()
}
