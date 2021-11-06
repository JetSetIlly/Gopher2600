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

// Package panel implements the front control panel of the VCS.
package panel

import (
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
)

// Panel represents the console's front control panel.
type Panel struct {
	id  plugging.PortID
	bus ports.PeripheralBus

	p0pro         bool
	p1pro         bool
	color         bool
	selectPressed bool
	resetPressed  bool
}

// NewPanel is the preferred method of initialisation for the Panel type.
func NewPanel(id plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	pan := &Panel{
		id:    id,
		bus:   bus,
		color: true,
	}
	pan.write()

	return pan
}

// Plumb implements the Peripheral interface.
func (pan *Panel) Plumb(bus ports.PeripheralBus) {
	pan.bus = bus
}

// String implements the Peripheral interface.
func (pan *Panel) String() string {
	s := strings.Builder{}

	s.WriteString("sel=")
	if pan.selectPressed {
		s.WriteString("held")
	} else {
		s.WriteString("no")
	}

	s.WriteString(", res=")
	if pan.resetPressed {
		s.WriteString("held")
	} else {
		s.WriteString("no")
	}

	s.WriteString(", p0=")
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

// PortID implements the ports.Peripheral interface.
func (pan *Panel) PortID() plugging.PortID {
	return pan.id
}

// ID implements the Peripheral interface.
func (pan *Panel) ID() plugging.PeripheralID {
	return plugging.PeriphPanel
}

// Reset implements the Peripheral interface.
func (pan *Panel) Reset() {
	// write current panel settings
	pan.write()
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

	pan.bus.WriteSWCHx(plugging.PortPanel, v)
}

// HandleEvent implements Peripheral interface.
func (pan *Panel) HandleEvent(event ports.Event, value ports.EventData) (bool, error) {
	var v bool
	switch d := value.(type) {
	case bool:
		v = d
	case ports.EventDataPlayback:
		if len(string(d)) > 0 {
			var err error
			v, err = strconv.ParseBool(string(d))
			if err != nil {
				return false, curated.Errorf("panel: %v: unexpected event data", event)
			}
		}
	}

	switch event {
	case ports.PanelSelect:
		pan.selectPressed = v

	case ports.PanelReset:
		pan.resetPressed = v

	case ports.PanelSetColor:
		pan.color = v

	case ports.PanelSetPlayer0Pro:
		pan.p0pro = v

	case ports.PanelSetPlayer1Pro:
		pan.p1pro = v

	case ports.PanelToggleColor:
		pan.color = !pan.color

	case ports.PanelTogglePlayer0Pro:
		pan.p0pro = !pan.p0pro

	case ports.PanelTogglePlayer1Pro:
		pan.p1pro = !pan.p1pro

	case ports.PanelPowerOff:
		return false, curated.Errorf(ports.PowerOff)

	default:
		return false, nil
	}

	pan.write()

	return true, nil
}

// Update implements the Peripheral interface.
func (pan *Panel) Update(data bus.ChipData) bool {
	return false
}

// Step implements the Peripheral interface.
func (pan *Panel) Step() {
}

// IsActive implements the ports.Peripheral interface.
func (pan *Panel) IsActive() bool {
	return false
}
