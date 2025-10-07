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

type pxe struct {
	// pxe is enabled if the pRAM symbol is defined
	enabled bool
	pRAM    uint32

	// pxe is only initialised once pRAM points to a valid address
	initialised bool
	origin      uint32

	// the last pxe palette address that was referenced. returns false if the PXE is not
	// enabled or initialised
	lastPaletteAddrQueue [256][]uint32
	lastPaletteAddr      [256]uint32
}

func (p *pxe) pushLastPaletteAddr(colour uint8, addr uint32) {
	p.lastPaletteAddrQueue[colour] = append(p.lastPaletteAddrQueue[colour], addr)
}
func (p *pxe) popLastPaletteAddr(colour uint8) (bool, uint32) {
	if len(p.lastPaletteAddrQueue[colour]) == 0 {
		v := p.lastPaletteAddr[colour]
		return v != 0, v
	}
	v := p.lastPaletteAddrQueue[colour][0]
	p.lastPaletteAddrQueue[colour] = p.lastPaletteAddrQueue[colour][1:]
	p.lastPaletteAddr[colour] = v
	return true, v
}

func (cart *Elf) LastPXEPalette(colour uint8) (bool, uint32) {
	ok, _ := cart.PXE()
	if !ok {
		return false, 0
	}
	return cart.mem.pxe.popLastPaletteAddr(colour)
}

// PXE returns true if a PXE section was found during loading. The returned uint32 value is the
// origin address of pRAM
func (cart *Elf) PXE() (bool, uint32) {
	if !cart.mem.pxe.enabled {
		return false, 0
	}

	if !cart.mem.pxe.initialised {
		var origin uint32
		if sec, ok := cart.mem.sectionsByName[pxeSection]; ok {
			origin, ok = cart.mem.Read32bit(sec.origin + cart.mem.pxe.pRAM)
			if !ok {
				return false, 0
			}

			// this isn't correct yet. we do this because we don't want to say PXE has been
			// initialised until we're sure origin is correct
			if origin == 0x00 {
				return true, 0
			}
		} else {
			return false, 0
		}

		cart.mem.pxe.initialised = true
		cart.mem.pxe.origin = origin
	}

	return true, cart.mem.pxe.origin
}
