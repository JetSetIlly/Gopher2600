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

package ports

import (
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// Panel represents the console's front control panel.
type Panel struct {
	bus PeripheralBus

	p0pro         bool
	p1pro         bool
	color         bool
	selectPressed bool
	resetPressed  bool
}

// NewPanel is the preferred method of initialisation for the Panel type.
func NewPanel(bus PeripheralBus) Peripheral {
	pan := &Panel{
		bus:   bus,
		color: true,
	}
	pan.write()

	return pan
}

// Plumb implements the Peripheral interface.
func (pan *Panel) Plumb(bus PeripheralBus) {
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

// Name implements the Peripheral interface.
func (pan *Panel) Name() string {
	return "Panel"
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

	pan.bus.WriteSWCHx(PanelID, v)
}

// Sentinal error returned by Panel.HandleEvent() if power button is pressed.
const (
	PowerOff = "emulated machine has been powered off"
)

// HandleEvent implements Peripheral interface.
func (pan *Panel) HandleEvent(event Event, value EventData) error {
	switch event {
	case PanelSelect:
		pan.selectPressed = value.(bool)

	case PanelReset:
		pan.resetPressed = value.(bool)

	case PanelSetColor:
		pan.color = value.(bool)

	case PanelSetPlayer0Pro:
		pan.p0pro = value.(bool)

	case PanelSetPlayer1Pro:
		pan.p1pro = value.(bool)

	case PanelToggleColor:
		pan.color = !pan.color

	case PanelTogglePlayer0Pro:
		pan.p0pro = !pan.p0pro

	case PanelTogglePlayer1Pro:
		pan.p1pro = !pan.p1pro

	case PanelPowerOff:
		return curated.Errorf(PowerOff)
	}

	pan.write()

	return nil
}

// Update implements the Peripheral interface.
func (pan *Panel) Update(data bus.ChipData) bool {
	return false
}

// Step implements the Peripheral interface.
func (pan *Panel) Step() {
}
