package digitalsticks

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/peripherals"

	"github.com/splace/joysticks"
)

// SplaceStick emulaes the digital VCS joystick
type SplaceStick struct {
	*peripherals.DigitalStick

	device *joysticks.HID
	err    error
}

// NewSplaceStick is the preferred method of initialisation for the Stick type
func NewSplaceStick(player int, tia memory.PeriphBus, riot memory.PeriphBus, panel *peripherals.Panel) (*SplaceStick, error) {
	var err error

	sps := new(SplaceStick)
	sps.DigitalStick, err = peripherals.NewDigitalStick(player, riot, tia)
	if err != nil {
		return nil, err
	}

	// there is a flaw (either in splace/joysticks or somewehere else lower
	// down in the kernel driver) which means that Connect() will not return
	// until it recieves some input from the controller. to get around this,
	// we've put the main body of the NewStick() function in a go routine.
	go func() {
		// try connecting to specific controller.
		// system assigned index: typically increments on each new controller added.
		sps.device = joysticks.Connect(1)
		if sps.device == nil {
			sps.err = errors.NewFormattedError(errors.NoControllerHardware, nil)
			return
		}

		// get/assign channels for specific events
		stickMove := sps.device.OnMove(1)

		buttonPress := sps.device.OnClose(1)
		buttonRelease := sps.device.OnOpen(1)

		// on xbox controller, button 8 is the start button
		resetPress := sps.device.OnClose(8)
		resetRelease := sps.device.OnOpen(8)

		// on xbox controller, button 9 is the back button
		selectPress := sps.device.OnClose(7)
		selectRelease := sps.device.OnOpen(7)

		// start feeding OS events onto the event channels.
		go sps.device.ParcelOutEvents()

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
				sps.DigitalStick.Handle("FIRE")
			case <-buttonRelease:
				sps.DigitalStick.Handle("NOFIRE")

			case ev := <-stickMove:
				x := ev.(joysticks.CoordsEvent).X
				y := ev.(joysticks.CoordsEvent).Y
				if x < -0.5 {
					sps.DigitalStick.Handle("LEFT")
				} else if x > 0.5 {
					sps.DigitalStick.Handle("RIGHT")
				} else if y < -0.5 {
					sps.DigitalStick.Handle("UP")
				} else if y > 0.5 {
					sps.DigitalStick.Handle("DOWN")
				} else {
					sps.DigitalStick.Handle("CENTRE")
				}
			}
		}
	}()

	return sps, nil
}

// Strobe implements the Controller interface
func (sps *SplaceStick) Strobe() error {
	return nil
}
