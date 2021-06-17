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

import "github.com/jetsetilly/gopher2600/logger"

// memory addressing module. not fully implemented.
type mam struct {
	mamcr          uint32
	mamtim         uint32
	allowFromThumb bool
}

// MAM addresses from UM10161 (page 20).
const (
	MAMCR  = PeripheralsOrigin | 0x001fc000
	MAMTIM = PeripheralsOrigin | 0x001fc004
)

func (m *mam) write(addr uint32, val uint32) bool {
	switch addr {
	case MAMCR:
		if m.allowFromThumb {
			m.mamcr = val
			if m.mamcr > 3 {
				logger.Logf("ARM7", "setting MAMCR to a value greater than 3 (%#08x)", m.mamcr)
			}
		}
	case MAMTIM:
		if m.allowFromThumb {
			if m.mamcr == 0 {
				m.mamtim = val
			} else {
				logger.Logf("ARM7", "trying to set MAMTIM while MAMCR is active")
			}
		}
	default:
		return false
	}

	return true
}

func (m *mam) read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case MAMCR:
		val = m.mamcr
	case MAMTIM:
		val = m.mamtim
	default:
		return 0, false
	}

	return val, true
}

func (m *mam) enable(enabled bool) {
	if enabled {
		m.mamcr = 1
	} else {
		m.mamcr = 0
	}
}

func (m *mam) isEnabled() bool {
	return m.mamcr != 0
}
