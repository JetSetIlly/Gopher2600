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

package rewind

// ComparisonState is returned by GetComparisonState()
type ComparisonState struct {
	State  *State
	Locked bool
}

// GetComparisonState gets a reference to current comparison point
func (r *Rewind) GetComparisonState() ComparisonState {
	return ComparisonState{
		State:  r.comparison.snapshot(),
		Locked: r.comparisonLocked,
	}
}

// UpdateComparison points comparison to the current state
func (r *Rewind) UpdateComparison() {
	if r.comparisonLocked {
		return
	}
	r.comparison = r.GetCurrentState()
}

// SetComparison points comparison to the supplied state
func (r *Rewind) SetComparison(frame int) {
	res := r.findFrameIndexExact(frame)
	s := r.entries[res.nearestIdx]
	if s != nil {
		r.comparison = s.snapshot()
	}
}

// LockComparison stops the comparison point from being updated
func (r *Rewind) LockComparison(locked bool) {
	r.comparisonLocked = locked
}
