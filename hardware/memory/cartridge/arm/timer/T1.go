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

// T1 implements a simple timer as used in the LCP2000.
type T1 struct {
	mmap    architecture.Map
	cycles  cycles
	enabled bool
	reset   bool
	control uint32
	counter uint32
}

func NewT1(mmap architecture.Map) *T1 {
	return &T1{
		mmap: mmap,
		cycles: cycles{
			clkDiv: mmap.ClkDiv,
		},
	}
}

// Reset implementes the Timer interface
func (t *T1) Reset() {
	t.cycles.reset()
}

// Step implementes the Timer interface
func (t *T1) Step(cycles float32) {
	if t.reset {
		t.cycles.reset()
	}
	if !t.enabled {
		return
	}
	if t.cycles.step(cycles) {
		t.counter += t.cycles.resolve()
	}
}

// Resolve implementes the Timer interface
func (t *T1) Resolve() {
	t.counter += t.cycles.resolve()
}

// Read implementes the Timer interface
func (t *T1) Read(addr uint32) (uint32, bool) {
	var val uint32

	switch addr {
	case t.mmap.T1TCR:
		// reserved bits could be randomised
		val = t.control
	case t.mmap.T1TC:
		t.Resolve()
		val = t.counter
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
		t.counter = val
	default:
		return false
	}

	return true
}
