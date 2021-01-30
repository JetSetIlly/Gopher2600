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

type timer struct {
	active bool

	control uint32
	counter float32
}

const (
	TIMERcontrol = PeripheralsOrigin | 0x00008004
	TIMERvalue   = PeripheralsOrigin | 0x00008008
)

func (t *timer) stepFromVCS(clock float32) {
	if !t.active {
		return
	}

	// the ARM timer ticks forward once every ARM cycle. the best we can do to
	// accommodate this is to tick the counter forward by the the appropriate
	// fraction every VCS cycle. Put another way: an NTSC spec VCS, for
	// example, will tick forward every 58-59 ARM cycles.
	t.counter += float32(InternalClk) / clock
}

func (t *timer) step(cycles float32) {
	if !t.active {
		return
	}
	t.counter += cycles
}

func (t *timer) write(addr uint32, val uint32) bool {
	switch addr {
	case TIMERcontrol:
		t.control = val
		t.active = t.control&0x01 == 0x01
	case TIMERvalue:
		t.counter = float32(val)
	default:
		return false
	}

	return true
}

func (t *timer) read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case TIMERcontrol:
		val = t.control
	case TIMERvalue:
		val = uint32(t.counter)
	default:
		return 0, false
	}

	return val, true
}
