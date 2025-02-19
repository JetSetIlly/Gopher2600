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

import "math"

// Timer is an abstraction of the various types of timer found in the difference
// ARM CPU pacakages
type Timer interface {
	Reset()
	Step(cycles float32)
	Resolve()
	Write(addr uint32, val uint32) bool
	Read(addr uint32) (uint32, bool)
}

type cycles struct {
	// number of accumulated cycles since the last reset() or resolve(). in the
	// case of resolve there may be a fractional amount of cycles remaining
	accumulation float32

	// number of calls to step
	stepCount int

	// timer devices use the peripheral clock (PCLK) rather than the clock of
	// the processor (CCLK) directly. the ClkDiv value scales the incoming
	// number cycles. we delay this to when we resolve() the timer
	clkDiv float32
}

// number of calls to step before the timer must be resolved
const resolveOnStepCount = 10000

func (t *cycles) reset() {
	t.accumulation = 0
}

func (t *cycles) step(cycles float32) bool {
	t.accumulation += cycles
	t.stepCount++
	if t.stepCount >= resolveOnStepCount {
		t.stepCount = 0
		return true
	}
	return false
}

func (t *cycles) resolve() uint32 {
	t.accumulation /= t.clkDiv
	i, f := math.Modf(float64(t.accumulation))
	t.accumulation = float32(f)
	return uint32(i)
}
