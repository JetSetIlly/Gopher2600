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

package peripherals

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// Timer implements a simple timer as used in the LCP2000.
type Timer struct {
	mmap    architecture.Map
	enabled bool
	reset   bool
	control uint32
	counter uint32
}

func NewTimer(mmap architecture.Map) Timer {
	return Timer{
		mmap: mmap,
	}
}

func (t *Timer) Reset() {
	t.counter = 0
}

// stepping of timer assumes an APB divider value of one.
func (t *Timer) Step(cycles uint32) {
	if t.reset {
		t.counter = 0
	}
	if !t.enabled {
		return
	}
	t.counter += cycles
}

func (t *Timer) Write(addr uint32, val uint32) bool {
	switch addr {
	case t.mmap.T1TCR:
		// from "5.2 Timer Control Register" in "UM10161", page 196
		t.control = val
		t.enabled = val&0x01 == 0x01
		t.reset = val&0x02 == 0x02
	case t.mmap.T1TC:
		t.counter = val
	default:
		return false
	}

	return true
}

func (t *Timer) Read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case t.mmap.T1TCR:
		// reserved bits could be randomised
		val = t.control
	case t.mmap.T1TC:
		val = t.counter
	default:
		return 0, false
	}

	return val, true
}
