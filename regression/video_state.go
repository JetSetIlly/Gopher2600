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

package regression

import (
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

// Indicates the State recording method to use.
type StateType string

// List of valid StateType values.
const (
	StateNone  StateType = ""
	StateTV    StateType = "TV"
	StatePorts StateType = "PORTS"
	StateTimer StateType = "TIMER"
	StateCPU   StateType = "CPU"
)

// NewStateType parses a string and returns a new StateType or an error. Use
// this rather than casting a string to the StateType.
func NewStateType(state string) (StateType, error) {
	switch strings.ToUpper(state) {
	case "":
		return StateNone, nil
	case "TV":
		return StateTV, nil
	case "PORTS":
		return StatePorts, nil
	case "TIMER":
		return StateTimer, nil
	case "CPU":
		return StateCPU, nil
	}
	return StateNone, curated.Errorf("regression: video: unrecognised state type [%s]", state)
}

func (t StateType) String() string {
	return string(t)
}
