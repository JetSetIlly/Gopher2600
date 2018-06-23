package controller

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"

	"github.com/splace/joysticks"
)

// Stick emulaes the digital VCS joystick
type Stick struct {
	device *joysticks.HID
	err    error
}

// NewStick is the preferred method of initialisation for the Stick type
func NewStick(tia *memory.ChipMemory) *Stick {
	stick := new(Stick)

	// there is a flaw (either in splace/joysticks or somewehere else lower
	// down in the kernel driver) which means that Connect() will not return
	// until it recieves some input from the controller. to get around this,
	// we've put the main body of the NewStick() function in a go routine.
	go func() {
		// try connecting to specific controller.
		// system assigned index: typically increments on each new controller added.
		stick.device = joysticks.Connect(1)
		if stick.device == nil {
			stick.err = errors.GopherError{errors.NoControllersFound, nil}
			return
		}

		// get/assign channels for specific events
		buttonPress := stick.device.OnClose(1)
		buttonRelease := stick.device.OnOpen(1)

		// start feeding OS events onto the event channels.
		go stick.device.ParcelOutEvents()

		// handle event channels
		for {
			select {
			case <-buttonPress:
				tia.ChipWrite("INPT4", 0x00)
			case <-buttonRelease:
				tia.ChipWrite("INPT4", 0x80)
			}
		}
	}()

	return stick
}
