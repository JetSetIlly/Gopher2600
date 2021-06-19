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
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/logger"
)

// memory addressing module. not fully implemented.
type mam struct {
	mmap MemoryMap

	// valid values for mamcr are 0, 1 or 2 are valid. we can think of these
	// respectively, as "disable", "partial" and "full"
	mamcr uint32

	// NOTE: not used yet
	mamtim uint32

	// the preference value
	pref int
}

func (m *mam) write(addr uint32, val uint32) bool {
	switch addr {
	case m.mmap.MAMCR:
		if m.pref == preferences.MAMDriver {
			m.setMAMCR(val)
		}
	case m.mmap.MAMTIM:
		if m.pref == preferences.MAMDriver {
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
	case m.mmap.MAMCR:
		val = m.mamcr
	case m.mmap.MAMTIM:
		val = m.mamtim
	default:
		return 0, false
	}

	return val, true
}

func (m *mam) setMAMCR(val uint32) {
	m.mamcr = val
	if m.mamcr > 2 {
		logger.Logf("ARM7", "setting MAMCR to a value greater than 2 (%#08x)", m.mamcr)
	}
}
