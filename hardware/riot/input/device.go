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

// ID differentiates the different devices attached to the console. note that
// PlayerZero and PlayerOne can have different types of devices attached to
// them (joysticks, paddles, keyboards)
type ID int

// List of defined IDs
const (
	PlayerZeroID ID = iota
	PlayerOneID
	PanelID
	NumIDs
)

type device struct {
	id     ID
	handle func(Event) error

	controller     Controller
	prevController Controller
	recorder       EventRecorder
}

// Attach a Controller implementation to the device. A Controller value of nil
// will reattach the previously attached controller. Futher calls with a value
// of nil will effectively remove all attached controllers. Events can still be
// pushed to the device by using the device's Handle() function directly.
func (dev *device) Attach(controller Controller) {
	if controller == nil {
		dev.controller = dev.prevController
		dev.prevController = nil
	} else {
		dev.prevController = dev.controller
		dev.controller = controller
	}
}

// AttachEventRecorder to the device. An EventRecorder value of nil will
// remove the recorder from the device.
func (dev *device) AttachEventRecorder(scribe EventRecorder) {
	dev.recorder = scribe
}

// CheckInput polls attached controllers for the most recent Event
func (dev *device) CheckInput() error {
	if dev.controller != nil {
		ev, err := dev.controller.CheckInput(dev.id)
		if err != nil {
			return err
		}

		err = dev.handle(ev)
		if err != nil {
			if !errors.Is(err, errors.InputDeviceUnplugged) {
				return err
			}
			dev.Attach(nil)
		}
	}

	return nil
}
