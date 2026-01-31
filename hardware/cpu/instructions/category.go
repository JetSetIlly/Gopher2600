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

package instructions

// Category of an instruction describes its effect
type Category int

const (
	Read Category = iota
	Write
	Modify
	Flow
	Subroutine
	Interrupt
)

func (e Category) String() string {
	switch e {
	case Read:
		return "Read"
	case Write:
		return "Write"
	case Modify:
		return "Modify"
	case Flow:
		return "Flow"
	case Subroutine:
		return "Subroutine"
	case Interrupt:
		return "Interrupt"
	}
	return "unknown effect"
}
