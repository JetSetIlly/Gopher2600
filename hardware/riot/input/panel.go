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

import (
	"strings"

	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/addresses"
)

// Panel represents the console's front control panel
type Panel struct {
	port

	mem *inputMemory

	p0pro         bool
	p1pro         bool
	color         bool
	selectPressed bool
	resetPressed  bool

	// data direction register
	ddr uint8
}

// NewPanel is the preferred method of initialisation for the Panel type
func NewPanel(mem *inputMemory) *Panel {
	pan := &Panel{
		mem:   mem,
		color: true,
	}

	pan.port = port{
		id:     PanelID,
		handle: pan.Handle,
	}

	pan.write()

	return pan
}

// String implements the Port interface
func (pan *Panel) String() string {
	s := strings.Builder{}

	s.WriteString("p0=")
	if pan.p0pro {
		s.WriteString("pro")
	} else {
		s.WriteString("am")
	}

	s.WriteString(", p1=")
	if pan.p1pro {
		s.WriteString("pro")
	} else {
		s.WriteString("am")
	}

	s.WriteString(", ")

	if pan.color {
		s.WriteString("col")
	} else {
		s.WriteString("b&w")
	}

	return s.String()
}

func (pan *Panel) write() {
	// commit changes to RIOT memory
	v := uint8(0)

	// pins 2, 4 and 5 are not used and always value value of 1
	v |= 0x20
	v |= 0x10
	v |= 0x04

	if pan.p0pro {
		v |= 0x80
	}

	if pan.p1pro {
		v |= 0x40
	}

	if pan.color {
		v |= 0x08
	}

	if !pan.selectPressed {
		v |= 0x02
	}

	if !pan.resetPressed {
		v |= 0x01
	}

	pan.mem.riot.InputDeviceWrite(addresses.SWCHB, v, pan.ddr)
}

// Handle implements Port interface
func (pan *Panel) Handle(event Event, value EventData) error {
	switch event {
	case PanelSelect:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}
		pan.selectPressed = b

	case PanelReset:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}
		pan.resetPressed = b

	case PanelSetColor:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}
		pan.color = b

	case PanelSetPlayer0Pro:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}
		pan.p0pro = b

	case PanelSetPlayer1Pro:
		b, ok := value.(bool)
		if !ok {
			return errors.New(errors.BadInputEventType, event, "bool")
		}
		pan.p1pro = b

	case PanelToggleColor:
		pan.color = !pan.color

	case PanelTogglePlayer0Pro:
		pan.p0pro = !pan.p0pro

	case PanelTogglePlayer1Pro:
		pan.p1pro = !pan.p1pro

	case PanelPowerOff:
		return errors.New(errors.PowerOff)

	case NoEvent:
		return nil

	default:
		return errors.New(errors.UnknownInputEvent, pan.id, event)
	}

	pan.write()

	// record event with the EventRecorder
	if pan.recorder != nil {
		return pan.recorder.RecordEvent(pan.id, event, value)
	}

	return nil
}
