// Panel uses the concurrent chip bus interface

package peripherals

import (
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
)

// Panel represents the console's front control panel
type Panel struct {
	riot  memory.PeriphBus
	p0pro bool
	p1pro bool
	color bool

	// select and reset switches do not toggle, they are triggered
	selectPressed bool
	resetPressed  bool
}

// NewPanel is the preferred method of initialisation for the Panel type
func NewPanel(riot memory.PeriphBus) *Panel {
	pan := new(Panel)
	pan.riot = riot
	pan.color = true
	pan.Strobe()
	return pan
}

// Strobe makes sure the panel has submitted its latest input
func (pan *Panel) Strobe() {
	strobe := uint8(0)

	// pins 2, 4 and 5 are not used and always value value of 1
	strobe |= 0x20
	strobe |= 0x10
	strobe |= 0x04

	if pan.p0pro {
		strobe |= 0x80
	}

	if pan.p1pro {
		strobe |= 0x40
	}

	if pan.color {
		strobe |= 0x08
	}

	if !pan.selectPressed {
		strobe |= 0x02
	}

	if !pan.resetPressed {
		strobe |= 0x01
	}

	pan.riot.PeriphWrite(vcssymbols.SWCHB, strobe)
}

// ToggleColour toggles the colour switch
func (pan *Panel) ToggleColour() {
	pan.color = !pan.color
}

// TogglePlayer0Pro toggles the color switch
func (pan *Panel) TogglePlayer0Pro() {
	pan.p0pro = !pan.p0pro
}

// TogglePlayer1Pro toggles the color switch
func (pan *Panel) TogglePlayer1Pro() {
	pan.p1pro = !pan.p1pro
}

// PressSelect emulates the select switch
func (pan *Panel) PressSelect() {
	pan.selectPressed = true
}

// PressReset emulates the reset switch
func (pan *Panel) PressReset() {
	pan.resetPressed = true
}

// ReleaseSelect emulates the select switch
func (pan *Panel) ReleaseSelect() {
	pan.selectPressed = false
}

// ReleaseReset emulates the reset switch
func (pan *Panel) ReleaseReset() {
	pan.resetPressed = false
}
