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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package input

import "gopher2600/errors"

// ID differentiates the different ports attached to the console
type ID int

// List of defined IDs
const (
	HandControllerZeroID ID = iota
	HandControllerOneID
	PanelID
	NumIDs
)

// Port conceptualises the I/O ports as described in the Stella Programmer's
// Guide (page 13). Port B is the VCS's switches on its front panel, while Port
// A allows the various hand controllers to be attached.
//
// See the HandController and Panel types for more information
type Port interface {
	String() string
	Handle(Event, EventValue) error
	AttachPlayback(Playback)
	AttachEventRecorder(EventRecorder)
	CheckInput() error
}

// port is the underlying commonality between all Port implementations
type port struct {
	id       ID
	playback Playback
	recorder EventRecorder
	handle   func(Event, EventValue) error
}

// Attach a Playback implementation to the port.  Events can still be
// pushed to the port by using the port's Handle() function directly. a
// Playback of nill will remove an existing playback from the port.
func (p *port) AttachPlayback(playback Playback) {
	p.playback = playback
}

// AttachEventRecorder to the port. An EventRecorder value of nil will
// remove the recorder from the port.
func (p *port) AttachEventRecorder(scribe EventRecorder) {
	p.recorder = scribe
}

// CheckInput polls the attached playback for an Event
func (p *port) CheckInput() error {
	if p.playback != nil {
		ev, v, err := p.playback.CheckInput(p.id)
		if err != nil {
			return err
		}

		err = p.handle(ev, v)
		if err != nil {
			if !errors.Is(err, errors.InputDeviceUnplugged) {
				return err
			}
			p.AttachPlayback(nil)
		}
	}

	return nil
}
