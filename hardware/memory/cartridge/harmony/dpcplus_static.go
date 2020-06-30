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

package harmony

import (
	"fmt"
	"strings"
)

// DPCplusStatic implements the bus.CartStatic interface
type DPCplusStatic struct {
	Arm  []byte
	Data []byte
	Freq []byte
}

func (sa DPCplusStatic) String() string {
	s := &strings.Builder{}

	// static data
	s.WriteString("Data    -0 -1 -2 -3 -4 -5 -6 -7 -8 -9 -A -B -C -D -E -F\n")
	s.WriteString("      ---- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --")

	j := uint16(0)
	for i := 0; i < len(sa.Data); i++ {
		// begin new row every 16 iterations
		if j%16 == 0 {
			s.WriteString(fmt.Sprintf("\n%03x- |  ", i/16))
		}
		s.WriteString(fmt.Sprintf("%02x ", sa.Data[i]))
		j++
	}

	// frequency table
	s.WriteString("\n\n")
	s.WriteString("Freq    -0 -1 -2 -3 -4 -5 -6 -7 -8 -9 -A -B -C -D -E -F\n")
	s.WriteString("      ---- -- -- -- -- -- -- -- -- -- -- -- -- -- -- --")

	j = uint16(0)
	for i := 0; i < len(sa.Freq); i++ {
		// begin new row every 16 iterations
		if j%16 == 0 {
			s.WriteString(fmt.Sprintf("\n%03x- |  ", i/16))
		}
		s.WriteString(fmt.Sprintf("%02x ", sa.Freq[i]))
		j++
	}

	return s.String()
}
