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

// Trace records the state of the electrical line, whether it is high or low, and also whether the
// immediately previous state is also high or low.
//
// Moving from one state to the other is done with Tick(bool) where a boolean value of true
// indicates a high voltage state.
//
// The function Falling() returns true if the line voltage has moved from a high state to low state;
// and Rising() returns true if the opposite is true.
//
// Deriving conditions from two traces is convenient. For example, given two traces A and B, a
// condition for event E might be:
//
//	 if A.Hi() && B.Rising() {
//			E()
//	 }
type Trace struct {
	Label string

	// new values are added to the end of the array
	Activity []bool

	from bool
	to   bool
}

const (
	activityLength = 64
)

func NewTrace(label string) Trace {
	tr := Trace{
		Label:    label,
		Activity: make([]bool, activityLength),
	}
	for i := range tr.Activity {
		tr.Activity[i] = true
	}
	return tr
}

func (tr *Trace) Snapshot() *Trace {
	cp := *tr
	cp.Activity = make([]bool, len(tr.Activity))
	copy(cp.Activity, tr.Activity)
	return &cp
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
	tr.Activity = append(tr.Activity[1:], v)
}
