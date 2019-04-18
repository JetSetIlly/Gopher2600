package peripherals

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
)

// A Port instance is used by controllers to communicate with the VCS
type Port struct {
	controller     Controller
	prevController Controller

	id string

	riot  memory.PeriphBus
	tia   memory.PeriphBus
	panel *Panel

	scribe Transcriber

	// joysticks
	joystick   uint16 // RIOT address
	fireButton uint16 // TIA address

	// poth player ports write to the same joystick address but in a slightly
	// different way. the joystickFunc allows an easy way to transform player0
	// values to player1 values
	joystickFunc func(uint8) uint8
}

// NewPlayer0 should be used to create a new communication port for
// controllers used by player 0
func NewPlayer0(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *Port {
	pt := &Port{
		id:           "Player0",
		riot:         riot,
		tia:          tia,
		panel:        panel,
		joystick:     vcssymbols.SWCHA,
		fireButton:   vcssymbols.INPT4,
		joystickFunc: func(n uint8) uint8 { return n }}

	pt.riot.PeriphWrite(pt.joystick, 0xff)
	pt.tia.PeriphWrite(pt.fireButton, 0x80)

	return pt
}

// NewPlayer1 should be used to create a new communication port for
// controllers used by player 1
func NewPlayer1(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *Port {
	pt := &Port{
		id:           "Player1",
		riot:         riot,
		tia:          tia,
		panel:        panel,
		joystick:     vcssymbols.SWCHA,
		fireButton:   vcssymbols.INPT5,
		joystickFunc: func(n uint8) uint8 { return n>>4 | 0xf0 }}

	pt.riot.PeriphWrite(pt.joystick, 0xff)
	pt.tia.PeriphWrite(pt.fireButton, 0x80)

	return pt
}

// Handle interprets an event into the correct sequence of memory addressing
func (pt Port) Handle(event Event) error {
	switch event {
	case Left:
		pt.riot.PeriphWrite(pt.joystick, pt.joystickFunc(0xbf))
	case Right:
		pt.riot.PeriphWrite(pt.joystick, pt.joystickFunc(0x7f))
	case Up:
		pt.riot.PeriphWrite(pt.joystick, pt.joystickFunc(0xef))
	case Down:
		pt.riot.PeriphWrite(pt.joystick, pt.joystickFunc(0xdf))
	case Centre:
		pt.riot.PeriphWrite(pt.joystick, pt.joystickFunc(0xff))
	case Fire:
		pt.tia.PeriphWrite(pt.fireButton, 0x00)
	case NoFire:
		pt.tia.PeriphWrite(pt.fireButton, 0x80)

	// for convenience, a controller implementation can interact with the panel
	case PanelSelectPress:
		pt.panel.PressSelect()
	case PanelSelectRelease:
		pt.panel.ReleaseSelect()
	case PanelResetPress:
		pt.panel.PressReset()
	case PanelResetRelease:
		pt.panel.ReleaseReset()

	case Unplugged:
		return errors.NewFormattedError(errors.ControllerUnplugged)

	// return now if there is no event to process
	default:
		return nil
	}

	// record event with the transcriber
	if pt.scribe != nil {
		return pt.scribe.Transcribe(pt.id, event)
	}

	return nil
}

// Attach registers a controller implementation with the port
func (pt *Port) Attach(controller Controller) {
	if controller == nil {
		pt.controller = pt.prevController
		pt.prevController = nil
	} else {
		pt.prevController = pt.controller
		pt.controller = controller
	}
}

// Strobe makes sure the controllers have submitted their latest input
func (pt *Port) Strobe() error {
	if pt.controller != nil {
		ev, err := pt.controller.GetInput(pt.id)
		if err != nil {
			return err
		}

		err = pt.Handle(ev)
		if err != nil {
			switch err := err.(type) {
			case errors.FormattedError:
				if err.Errno != errors.ControllerUnplugged {
					return err
				}
				pt.controller = pt.prevController
			default:
				return err
			}
		}
	}

	return nil
}

// AttachScribe registers the presence of a transcriber implementation. use an
// argument of nil to disconnect an existing scribe
func (pt *Port) AttachScribe(scribe Transcriber) {
	pt.scribe = scribe
}
