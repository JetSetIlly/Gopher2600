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

	// select and reset switches do not toggle. calling functions to
	// SetGameSelect() and SetGameReset() should emulate as best as possible
	gameSelect bool
	gameReset  bool
}

// NewPanel is the preferred method of initialisation for the Panel type
func NewPanel(riot memory.PeriphBus) *Panel {
	pan := new(Panel)
	pan.riot = riot
	pan.color = true
	pan.set()
	return pan
}

func (pan *Panel) set() {
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
	if !pan.gameSelect {
		strobe |= 0x02
	}
	if !pan.gameReset {
		strobe |= 0x01
	}

	pan.riot.PeriphWrite(vcssymbols.SWCHB, strobe)
}

// SetColor toggles the color switch
func (pan *Panel) SetColor(set bool) {
	pan.color = set
	pan.set()
}

// SetPlayer0Pro toggles the color switch
func (pan *Panel) SetPlayer0Pro(set bool) {
	pan.p0pro = set
	pan.set()
}

// SetPlayer1Pro toggles the color switch
func (pan *Panel) SetPlayer1Pro(set bool) {
	pan.p1pro = set
	pan.set()
}

// SetGameSelect toggles the color switch
func (pan *Panel) SetGameSelect(set bool) {
	pan.gameSelect = set
	pan.set()
}

// SetGameReset toggles the color switch
func (pan *Panel) SetGameReset(set bool) {
	pan.gameReset = set
	pan.set()
}
