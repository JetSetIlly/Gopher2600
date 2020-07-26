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

package future

// Observer exposes only the function relating to the observing of events
type Observer interface {
	Observe(label string) (*Event, bool)
}

// Observe looks for the most recent event with the specified label. if it is
// found then it is returned along with the value true to indicate a match. if
// it is not found, then the most recent event (with whatever label) is
// returned along with false to indicate no match
func (tck Ticker) Observe(label string) (*Event, bool) {
	// start from the back of the active list. i.e the entry just in front of
	// the active sentinal
	e := tck.activeSentinal.Prev()
	for e != nil {
		v := e.Value.(*Event)

		// return match
		if v.label == label && v.remainingCycles > -1 {
			return v, true
		}

		e = e.Prev()
	}

	// return most recent event if no match found
	e = tck.activeSentinal.Prev()
	if e == nil {
		return nil, false
	}

	return e.Value.(*Event), false
}
