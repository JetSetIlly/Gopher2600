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

// ResetValue is used to reset the polycounter.
const ResetValue = 0

// Polycounter counts through the entries of a 6 bit polycounter. For the
// purposes of the emulation we represent it as an integer and index a
// pre-calculated table as required.
type Polycounter int

func (p Polycounter) String() string {
	return fmt.Sprintf("%s (%02d)", p.ToBinary(), p)
}

// ToBinary returns the bit pattern of the current polycounter value.
func (p *Polycounter) ToBinary() string {
	return table6bit[*p]
}
