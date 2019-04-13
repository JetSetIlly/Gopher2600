package peripherals

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
	"strings"

	"github.com/splace/joysticks"
)

// Stick emulaes the digital VCS joystick
type Stick struct {
	device *joysticks.HID
	err    error

	tia  memory.PeriphBus
	riot memory.PeriphBus
}

// NewStick is the preferred method of initialisation for the Stick type
func NewStick(tia memory.PeriphBus, riot memory.PeriphBus, panel *Panel) *Stick {
	stk := new(Stick)
	stk.tia = tia
	stk.riot = riot

	// TODO: make all this work with a second contoller. for now, initialise
	// and asssume that there is just one controller for player 0
	stk.riot.PeriphWrite(vcssymbols.SWCHA, 0xff)
	stk.tia.PeriphWrite(vcssymbols.INPT4, 0x80)
	stk.tia.PeriphWrite(vcssymbols.INPT5, 0x80)

	// there is a flaw (either in splace/joysticks or somewehere else lower
	// down in the kernel driver) which means that Connect() will not return
	// until it recieves some input from the controller. to get around this,
	// we've put the main body of the NewStick() function in a go routine.
	go func() {
		// try connecting to specific controller.
		// system assigned index: typically increments on each new controller added.
		stk.device = joysticks.Connect(1)
		if stk.device == nil {
			stk.err = errors.NewFormattedError(errors.NoControllersFound, nil)
			return
		}

		// get/assign channels for specific events
		stickMove := stk.device.OnMove(1)

		buttonPress := stk.device.OnClose(1)
		buttonRelease := stk.device.OnOpen(1)

		// on xbox controller, button 8 is the start button
		resetPress := stk.device.OnClose(8)
		resetRelease := stk.device.OnOpen(8)

		// on xbox controller, button 9 is the back button
		selectPress := stk.device.OnClose(7)
		selectRelease := stk.device.OnOpen(7)

		// start feeding OS events onto the event channels.
		go stk.device.ParcelOutEvents()

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
				stk.HandleStick(0, "FIRE")
			case <-buttonRelease:
				stk.HandleStick(0, "NOFIRE")

			case ev := <-stickMove:
				x := ev.(joysticks.CoordsEvent).X
				y := ev.(joysticks.CoordsEvent).Y
				if x < -0.5 {
					stk.HandleStick(0, "LEFT")
				} else if x > 0.5 {
					stk.HandleStick(0, "RIGHT")
				} else if y < -0.5 {
					stk.HandleStick(0, "UP")
				} else if y > 0.5 {
					stk.HandleStick(0, "DOWN")
				} else {
					stk.HandleStick(0, "CENTRE")
				}
			}
		}
	}()

	return stk
}

// HandleStick parses the action and writes to the correct memory location
func (stk *Stick) HandleStick(player int, action string) error {
	var stickAddress uint16
	var fireAddress uint16

	if player == 0 {
		stickAddress = vcssymbols.SWCHA
		fireAddress = vcssymbols.INPT4
	} else if player == 1 {
		stickAddress = vcssymbols.SWCHB
		fireAddress = vcssymbols.INPT5
	} else {
		panic(fmt.Sprintf("there is no player %d with a joystick to handle", player))
	}

	switch strings.ToUpper(action) {
	case "LEFT":
		stk.riot.PeriphWrite(stickAddress, 0xbf)
	case "RIGHT":
		stk.riot.PeriphWrite(stickAddress, 0x7f)
	case "UP":
		stk.riot.PeriphWrite(stickAddress, 0xef)
	case "DOWN":
		stk.riot.PeriphWrite(stickAddress, 0xdf)
	case "CENTER":
		fallthrough
	case "CENTRE":
		stk.riot.PeriphWrite(stickAddress, 0xff)
	case "FIRE":
		stk.tia.PeriphWrite(fireAddress, 0x00)
	case "NOFIRE":
		stk.tia.PeriphWrite(fireAddress, 0x80)
	}

	return nil
}
