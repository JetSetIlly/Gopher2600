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

package timer

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/arm/architecture"
)

// T1 implements a simple timer as used in the LPC2000.
//
// An example of a game that uses T1 is Draconian. It uses it to test what spec
// the console is (NTSC, PAL, etc.)
//
// Another good example is Andrew Davie's ARM powered Boulderdash. it also uses
// the timer to test for the console spec; and also to time the wipe that starts
// and ends the level. character movemement is also affected by the timer
type T1 struct {
	mmap architecture.Map

	// storing the counter register as a float because it makes cycle counting
	// easier.  the value is truncated to a uint32 only when the T1TC register is read
	counter float32

	// the enabled and reset fields reflect the corresponding bits in the control register
	control uint32
	enabled bool
	reset   bool
}

func NewT1(mmap architecture.Map) *T1 {
	return &T1{
		mmap: mmap,
	}
}

// Reset implementes the Timer interface
func (t *T1) Reset() {
	t.counter = 0
}

// Step implementes the Timer interface
func (t *T1) Step(cycles float32) {
	if t.reset {
		// not setting reset flag to false because, "The counters remain reset until TCR[1] is
		// returned to zero" from "5.2 Timer Control Register" in "UM10161", page 197
		t.Reset()
	}
	if !t.enabled {
		return
	}
	t.counter += cycles
}

// Read implementes the Timer interface
func (t *T1) Read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case t.mmap.T1TCR:
		// TODO: reserved bits could be randomised
		val = t.control
	case t.mmap.T1TC:
		val = uint32(t.counter / t.mmap.ClkDiv)
	default:
		return 0, false
	}

	return val, true
}

// Write implementes the Timer interface
func (t *T1) Write(addr uint32, val uint32) bool {
	switch addr {
	case t.mmap.T1TCR:
		// from "5.2 Timer Control Register" in "UM10161", page 196
		t.control = val
		t.enabled = val&0x01 == 0x01
		t.reset = val&0x02 == 0x02
	case t.mmap.T1TC:
		t.counter = float32(val)
	default:
		return false
	}

	return true
}
