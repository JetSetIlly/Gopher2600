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
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Playback implementations feed controller Events to the device on request
// with the CheckInput() function.
//
// Intended for playback of controller events previously recorded to a file on
// disk but usable for many purposes I suspect. For example, AI control.
type EventPlayback interface {
	// note the type restrictions on EventData in the type definition's
	// commentary
	GetPlayback() (ports.TimedInputEvent, error)
}

// EventRecorder implementations mirror an incoming event.
//
// Implementations should be able to handle being attached to more than one
// peripheral at once. The ID parameter of the EventRecord() function will help
// to differentiate between multiple devices.
type EventRecorder interface {
	RecordEvent(ports.TimedInputEvent) error
}

// AttachEventRecorder attaches an EventRecorder implementation.
func (inp *Input) AddRecorder(r EventRecorder) {
	inp.recorder = append(inp.recorder, r)
}

// ClearRecorders removes all registered event recorders.
func (inp *Input) ClearRecorders() {
	inp.recorder = inp.recorder[:0]
}

// AttachPlayback attaches an EventPlayback implementation to the Input
// sub-system. EventPlayback can be nil in order to remove the playback.
func (inp *Input) AttachPlayback(pb EventPlayback) error {
	// we have previously checked whether a recorder was attached before
	// allowing playback. however, we are now allowing multiple recorders some
	// of which will never be replayed. moreover, it was never really clear
	// whether recording a playback file would be an issue. On reflection, I
	// don't think it would be - but it hasn't been tested
	inp.playback = pb
	inp.setHandleFunc()
	return nil
}

// handlePlaybackEvents requests playback events from all attached and eligible peripherals.
func (inp *Input) handlePlaybackEvents() error {
	if inp.playback == nil {
		return nil
	}

	// loop with GetPlayback() until we encounter a NoPortID or NoEvent
	// condition. there might be more than one entry for a particular
	// frame/scanline/horizpas state so we need to make sure we've processed
	// them all.
	//
	// this happens in particular with recordings that were made of  ROMs with
	// panel setup configurations (see setup package) - where the switches are
	// set when the TV state is at fr=0 sl=0 cl=0
	morePlayback := true
	for morePlayback {
		ev, err := inp.playback.GetPlayback()
		if err != nil {
			return err
		}

		morePlayback = ev.Port != plugging.PortUnplugged && ev.Ev != ports.NoEvent
		if morePlayback {
			_, err := inp.ports.HandleInputEvent(ev.InputEvent)
			if err != nil {
				return err
			}

			// forward to passenger if necessary
			if inp.toPassenger != nil {
				select {
				case inp.toPassenger <- ev:
				default:
					return fmt.Errorf("input: passenger event queue is full: input dropped")
				}
			}

			// forward event to attached recorders
			for _, r := range inp.recorder {
				err := r.RecordEvent(ev)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
