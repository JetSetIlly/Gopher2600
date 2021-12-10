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
func (inp *Input) AttachRecorder(r EventRecorder) error {
	if inp.playback != nil {
		return curated.Errorf("input: attach recorder: emulator already has a playback attached")
	}
	inp.recorder = r
	return nil
}

// AttachPlayback attaches an EventPlayback implementation to the Input
// sub-system. EventPlayback can be nil in order to remove the playback.
func (inp *Input) AttachPlayback(pb EventPlayback) error {
	if inp.recorder != nil {
		return curated.Errorf("input: attach playback: emulator already has a recorder attached")
	}
	inp.playback = pb
	inp.setProcessFunc()
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
					return curated.Errorf("input: passenger event queue is full: input dropped")
				}
			}
		}
	}

	return nil
}
