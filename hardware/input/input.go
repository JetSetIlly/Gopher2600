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
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// TV defines the television functions required by the Input system.
type TV interface {
	GetCoords() coords.TelevisionCoords
}

// Input handles all forms of input into the VCS.
type Input struct {
	tv    TV
	ports *ports.Ports

	playback EventPlayback
	recorder EventRecorder

	// events pushed onto the input queue
	pushed chan ports.InputEvent

	// the following fields all relate to driven input, for either the driver
	// or for the passenger (the driven)
	fromDriver       chan ports.TimedInputEvent
	toPassenger      chan ports.TimedInputEvent
	checkForDriven   bool
	drivenInputEvent ports.TimedInputEvent

	// Process function should be called every VCS step
	Process func() error
}

func NewInput(tv TV, p *ports.Ports) *Input {
	inp := &Input{
		tv:     tv,
		ports:  p,
		pushed: make(chan ports.InputEvent, 64),
	}
	inp.setProcessFunc()
	return inp
}

// Plumb a new ports instances into the Input.
func (inp *Input) Plumb(ports *ports.Ports) {
	inp.ports = ports
}

// PeripheralID forwards a request of the PeripheralID of the PortID to VCS Ports.
func (inp *Input) PeripheralID(id plugging.PortID) plugging.PeripheralID {
	return inp.ports.PeripheralID(id)
}

// HandleInputEvent forwards an input event to VCS Ports.
//
// If a playback is currently active the input will not be handled and false
// will be returned.
func (inp *Input) HandleInputEvent(ev ports.InputEvent) (bool, error) {
	if inp.playback != nil {
		return false, nil
	}

	if inp.recorder != nil {
		err := inp.recorder.RecordEvent(ports.TimedInputEvent{Time: inp.tv.GetCoords(), InputEvent: ev})
		if err != nil {
			return false, err
		}
	}

	handled, err := inp.ports.HandleInputEvent(ev)
	if err != nil {
		return handled, err
	}

	// forward to passenger if one is defined
	if handled && inp.toPassenger != nil {
		select {
		case inp.toPassenger <- ports.TimedInputEvent{Time: inp.tv.GetCoords(), InputEvent: ev}:
		default:
			return handled, curated.Errorf("input: passenger event queue is full: input dropped")
		}
	}

	return handled, nil
}

func (inp *Input) setProcessFunc() {
	if inp.fromDriver != nil && inp.playback != nil {
		inp.Process = func() error {
			if err := inp.handlePushedEvents(); err != nil {
				return err
			}
			if err := inp.handlePlaybackEvents(); err != nil {
				return err
			}
			if err := inp.handleDrivenEvents(); err != nil {
				return err
			}
			return nil
		}
		return
	}

	if inp.fromDriver != nil {
		inp.Process = func() error {
			if err := inp.handlePushedEvents(); err != nil {
				return err
			}
			if err := inp.handleDrivenEvents(); err != nil {
				return err
			}
			return nil
		}
		return
	}

	if inp.playback != nil {
		inp.Process = func() error {
			if err := inp.handlePushedEvents(); err != nil {
				return err
			}
			if err := inp.handlePlaybackEvents(); err != nil {
				return err
			}
			return nil
		}
		return
	}

	inp.Process = inp.handlePushedEvents
}
