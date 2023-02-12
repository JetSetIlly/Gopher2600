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

package i2c

// length of activity trace
const activityLength = 64

// Trace records the state of electrical line, whether it is high or low, and
// also whether the immediately previous state is also high or low.
//
// moving from one state to the other is done with tick(bool) where a boolean
// value of true indicates a high voltage state.
//
// the function hi2lo() returns true if the line voltage has moved from a high
// state to low state; and low2hi() returns true if the opposite is true.
//
// deriving conditions from two traces is convenient. for example, give two
// traces A and B, a condition for event E might be:
//
//	 if A.hi() && B.lo2hi() {
//			E()
//	 }
type Trace struct {
	// a recent history of the i2c trace. wraps around at activityLength
	activity []float32

	// ptr is the next activity index to be written to
	ptr int

	// rather than indexing the activity field and dealing with wrap around, we
	// record the hi state of the trace as a boolean for the most recent recent
	// time period and for the period before that (in the 2600 the period is
	// the speed of the 6507)
	//
	// labelled from and to because this is how we think about these values
	// when checking whether the trace is rising or falling
	from bool
	to   bool // most recent time period
}

const (
	TraceHi = 1.0
	TraceLo = -1.0
)

func NewTrace() Trace {
	tr := Trace{
		activity: make([]float32, activityLength),
	}
	for i := range tr.activity {
		tr.activity[i] = TraceHi
	}
	return tr
}

func (tr *Trace) Changed() bool {
	return tr.from != tr.to
}

func (tr *Trace) Falling() bool {
	return tr.from && !tr.to
}

func (tr *Trace) Rising() bool {
	return !tr.from && tr.to
}

func (tr *Trace) Hi() bool {
	return tr.to
}

func (tr *Trace) Lo() bool {
	return !tr.to
}

func (tr *Trace) Tick(v bool) {
	tr.from = tr.to
	tr.to = v
	if v {
		tr.activity[tr.ptr] = TraceHi
	} else {
		tr.activity[tr.ptr] = TraceLo
	}
	tr.ptr++
	if tr.ptr >= len(tr.activity) {
		tr.ptr = 0
	}
}

// Copy makes a copy of the activity trace.
func (tr *Trace) Copy() []float32 {
	c := make([]float32, len(tr.activity))

	copy(c, tr.activity[tr.ptr:])
	copy(c[len(tr.activity)-tr.ptr:], tr.activity[:tr.ptr])

	return c
}
