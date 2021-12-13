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

package input

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// handleDrivenEvents checks for input driven from a another emulation.
func (inp *Input) handleDrivenEvents() error {
	if inp.checkForDriven {
		ev := inp.drivenInputEvent
		done := false
		for !done {
			c := inp.tv.GetCoords()
			if coords.Equal(c, ev.Time) {
				_, err := inp.ports.HandleInputEvent(ev.InputEvent)
				if err != nil {
					return err
				}
			} else if coords.GreaterThan(c, ev.Time) {
				return curated.Errorf("input: driven input seen too late. emulations not synced correctly.")
			} else {
				return nil
			}

			select {
			case inp.drivenInputEvent = <-inp.fromDriver:
				if inp.checkForDriven {
					curated.Errorf("input: driven input received before previous input was processed")
				}
			default:
				done = true
				inp.checkForDriven = false
			}

			ev = inp.drivenInputEvent
		}
	}

	if inp.fromDriver != nil {
		select {
		case inp.drivenInputEvent = <-inp.fromDriver:
			if inp.checkForDriven {
				curated.Errorf("input: driven input received before previous input was processed")
			}
			inp.checkForDriven = true
		default:
		}
	}

	return nil
}

// AttachPassenger should be called by an emulation that wants to be driven by another emulation.
func (inp *Input) AttachPassenger(driver chan ports.TimedInputEvent) error {
	if inp.toPassenger != nil {
		return curated.Errorf("input: attach passenger: emulation already defined as an input driver")
	}
	inp.fromDriver = driver
	inp.setHandleFunc()
	return nil
}

// AttachDriver should be called by an emulation that is prepared to drive another emulation.
func (inp *Input) AttachDriver(passenger chan ports.TimedInputEvent) error {
	if inp.fromDriver != nil {
		return curated.Errorf("input: attach driver: emulation already defined as being an input passenger")
	}
	inp.toPassenger = passenger
	return nil
}
