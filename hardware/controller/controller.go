package controller

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
	"gopher2600/hardware/panel"

	"github.com/splace/joysticks"
)

// Stick emulaes the digital VCS joystick
type Stick struct {
	device *joysticks.HID
	err    error
}

// NewStick is the preferred method of initialisation for the Stick type
func NewStick(tia memory.ChipBus, riot memory.ChipBus, panel *panel.Panel) *Stick {
	stick := new(Stick)

	// TODO: make all this work with a seconc contoller. for now, initialise
	// and asssume that there is just one controller for player 0
	riot.ChipWrite(vcssymbols.SWCHA, 0xff)
	tia.ChipWrite(vcssymbols.INPT4, 0x80)
	tia.ChipWrite(vcssymbols.INPT5, 0x80)

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
		stickMove := stick.device.OnMove(1)

		buttonPress := stick.device.OnClose(1)
		buttonRelease := stick.device.OnOpen(1)

		// on xbox controller, button 8 is the start button
		resetPress := stick.device.OnClose(8)
		resetRelease := stick.device.OnOpen(8)

		// on xbox controller, button 9 is the back button
		selectPress := stick.device.OnClose(7)
		selectRelease := stick.device.OnOpen(7)

		// start feeding OS events onto the event channels.
		go stick.device.ParcelOutEvents()

		// handle event channels
		for {
			select {
			case <-resetPress:
				panel.SetGameReset(true)
			case <-resetRelease:
				panel.SetGameReset(false)

			case <-selectPress:
				panel.SetGameSelect(true)
			case <-selectRelease:
				panel.SetGameSelect(false)

			case <-buttonPress:
				tia.ChipWrite(vcssymbols.INPT4, 0x00)
			case <-buttonRelease:
				tia.ChipWrite(vcssymbols.INPT4, 0x80)

			case ev := <-stickMove:
				swcha := uint8(0xff)
				x := ev.(joysticks.CoordsEvent).X
				y := ev.(joysticks.CoordsEvent).Y
				if x < -0.5 {
					swcha &= 0xbf
				} else if x > 0.5 {
					swcha &= 0x7f
				}
				if y < -0.5 {
					swcha &= 0xef
				} else if y > 0.5 {
					swcha &= 0xdf
				}
				riot.ChipWrite(vcssymbols.SWCHA, swcha)
			}
		}
	}()

	return stick
}
