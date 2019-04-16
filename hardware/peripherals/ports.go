package peripherals

import (
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
)

// A Port instance is used by controllers to communicate with the VCS
type Port struct {
	cont Controller

	riot  memory.PeriphBus
	tia   memory.PeriphBus
	panel *Panel

	// joysticks
	joystick   uint16 // RIOT address
	fireButton uint16 // TIA address
}

// NewPlayer0 should be used to create a new communication port for
// controllers used by player 0
func NewPlayer0(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *Port {
	pt := &Port{
		riot:       riot,
		tia:        tia,
		panel:      panel,
		joystick:   vcssymbols.SWCHA,
		fireButton: vcssymbols.INPT4}

	pt.riot.PeriphWrite(vcssymbols.SWCHA, 0xff)
	pt.tia.PeriphWrite(vcssymbols.INPT4, 0x80)

	return pt
}

// NewPlayer1 should be used to create a new communication port for
// controllers used by player 1
func NewPlayer1(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *Port {
	pt := &Port{
		riot:       riot,
		tia:        tia,
		panel:      panel,
		joystick:   vcssymbols.SWCHB,
		fireButton: vcssymbols.INPT5}

	pt.riot.PeriphWrite(vcssymbols.SWCHB, 0xff)
	pt.tia.PeriphWrite(vcssymbols.INPT5, 0x80)

	return pt
}

// Handle interprets an event into the correct sequence of memory addressing
func (pt Port) Handle(event Event) error {
	switch event {
	case Left:
		pt.riot.PeriphWrite(pt.joystick, 0xbf)
	case Right:
		pt.riot.PeriphWrite(pt.joystick, 0x7f)
	case Up:
		pt.riot.PeriphWrite(pt.joystick, 0xef)
	case Down:
		pt.riot.PeriphWrite(pt.joystick, 0xdf)
	case Centre:
		pt.riot.PeriphWrite(pt.joystick, 0xff)
	case Fire:
		pt.tia.PeriphWrite(pt.fireButton, 0x00)
	case NoFire:
		pt.tia.PeriphWrite(pt.fireButton, 0x80)

	// for convenience, a controller implementation can interact with the panel

	case PanelSelectPress:
		pt.panel.SetGameSelect(true)
	case PanelResetPress:
		pt.panel.SetGameReset(true)
	case PanelSelectRelease:
		pt.panel.SetGameSelect(false)
	case PanelResetRelease:
		pt.panel.SetGameReset(false)
	}

	return nil
}

// Attach registers a controller implementation with the port
func (pt *Port) Attach(controller Controller) {
	pt.cont = controller
}

// Strobe makes sure the controllers have submitted their latest input
func (pt *Port) Strobe() {
	if pt.cont != nil {
		pt.Handle(pt.cont.GetInput())
	}
}
