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

// PXE related address values. All these are relative to the origin value returned
// by the PXE() function
const (
	PXEPaletteOrigin = 0x00000700
	PXEPaletteMemtop = 0x000007ff
	PXEMemtop        = 0x00000fff
)

// PXE returns true if a PXE section was found during loading. The returned uint32 value is the
// origin address of pRAM
func (cart *Elf) PXE() (bool, uint32) {
	var origin uint32
	if sec, ok := cart.mem.sectionsByName[pxeSection]; ok {
		origin, ok = cart.mem.Read32bit(sec.origin + cart.mem.pRAM)
		if !ok {
			return false, 0
		}
	} else {
		return false, 0
	}
	return cart.mem.hasPXE, origin
}
