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

package arm7tdmi

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/logger"
)

type scratch map[uint32]byte

// some scratch memory addresses are expected. we don't want to log these.
const (
	memAccelerationAddr = 0xe01fc000
)

func (scr *scratch) read8bit(addr uint32) uint8 {
	if addr != memAccelerationAddr {
		logger.Log("ARM7", fmt.Sprintf("reading %02x from %08x", (*scr)[addr], addr))
	}
	return (*scr)[addr]
}

func (scr *scratch) write8bit(addr uint32, val uint8) {
	if addr != memAccelerationAddr {
		logger.Log("ARM7", fmt.Sprintf("writing %02x to %08x", val, addr))
	}
	(*scr)[addr] = val
}

func (scr *scratch) read16bit(addr uint32) uint16 {
	v := uint16((*scr)[addr]) | (uint16((*scr)[addr+1]) << 8)
	if addr != memAccelerationAddr {
		logger.Log("ARM7", fmt.Sprintf("reading %04x from %08x", v, addr))
	}
	return v
}

func (scr *scratch) write16bit(addr uint32, val uint16) {
	if addr != memAccelerationAddr {
		logger.Log("ARM7", fmt.Sprintf("writing %04x to %08x", val, addr))
	}
	(*scr)[addr] = uint8(val)
	(*scr)[addr+1] = uint8(val >> 8)
}

func (scr *scratch) read32bit(addr uint32) uint32 {
	v := uint32((*scr)[addr]) | (uint32((*scr)[addr+1]) << 8) | (uint32((*scr)[addr+2]) << 16) | (uint32((*scr)[addr+3]) << 24)
	if addr != memAccelerationAddr {
		logger.Log("ARM7", fmt.Sprintf("reading %04x from %08x", v, addr))
	}
	return v
}

func (scr *scratch) write32bit(addr uint32, val uint32) {
	if addr != memAccelerationAddr {
		logger.Log("ARM7", fmt.Sprintf("writing %08x to %08x", val, addr))
	}
	(*scr)[addr] = uint8(val)
	(*scr)[addr+1] = uint8(val >> 8)
	(*scr)[addr+2] = uint8(val >> 16)
	(*scr)[addr+3] = uint8(val >> 24)
}
