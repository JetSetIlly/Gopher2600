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
//  if A.hi() && B.lo2hi() {
//		E()
//  }
type Trace struct {
	activity []float32
}

const (
	TraceHi = 1.0
	TraceLo = -1.0
)

func NewTrace() Trace {
	tr := Trace{
		activity: make([]float32, 64),
	}
	for i := range tr.activity {
		tr.activity[i] = TraceHi
	}
	return tr
}

func (tr *Trace) Recent() (from bool, to bool) {
	return tr.activity[len(tr.activity)-2] > 0, tr.activity[len(tr.activity)-1] > 0
}

func (tr *Trace) Changed() bool {
	from, to := tr.Recent()
	return from != to
}

func (tr *Trace) Falling() bool {
	from, to := tr.Recent()
	return from && !to
}

func (tr *Trace) Rising() bool {
	from, to := tr.Recent()
	return !from && to
}

func (tr *Trace) Hi() bool {
	from, _ := tr.Recent()
	return from
}

func (tr *Trace) Lo() bool {
	from, _ := tr.Recent()
	return !from
}

func (tr *Trace) Tick(v bool) {
	if v {
		tr.activity = append(tr.activity, TraceHi)
	} else {
		tr.activity = append(tr.activity, TraceLo)
	}
	tr.activity = tr.activity[1:]
}

// Copy makes a copy of the activity trace.
func (tr *Trace) Copy() []float32 {
	c := make([]float32, len(tr.activity))
	copy(c, tr.activity)
	return c
}
