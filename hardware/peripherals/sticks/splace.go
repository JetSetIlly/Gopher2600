package sticks

import (
	"gopher2600/errors"
	"gopher2600/hardware/peripherals"

	"github.com/splace/joysticks"
)

// SplaceStick emulaes the digital VCS joystick
type SplaceStick struct {
	event chan peripherals.Event

	device *joysticks.HID
	err    error
}

// NewSplaceStick is the preferred method of initialisation for the Stick type
func NewSplaceStick() (*SplaceStick, error) {
	sps := new(SplaceStick)
	sps.event = make(chan peripherals.Event)

	go func() {
		// try connecting to specific controller.
		// system assigned index: typically increments on each new controller added.
		sps.device = joysticks.Connect(1)
		if sps.device == nil {
			sps.err = errors.NewFormattedError(errors.NoControllerHardware, nil)
			return
		}

		// create channels for specific events
		stickMove := sps.device.OnMove(1)
		buttonPress := sps.device.OnClose(1)
		buttonRelease := sps.device.OnOpen(1)
		resetPress := sps.device.OnClose(8) // start button
		resetRelease := sps.device.OnOpen(8)
		selectPress := sps.device.OnClose(7) // back button
		selectRelease := sps.device.OnOpen(7)

		// start feeding OS events onto the event channels.
		go sps.device.ParcelOutEvents()

		// handle event channels
		for {
			select {
			case ev := <-stickMove:
				x := ev.(joysticks.CoordsEvent).X
				y := ev.(joysticks.CoordsEvent).Y
				if x < -0.5 {
					sps.event <- peripherals.Left
				} else if x > 0.5 {
					sps.event <- peripherals.Right
				} else if y < -0.5 {
					sps.event <- peripherals.Up
				} else if y > 0.5 {
					sps.event <- peripherals.Down
				} else {
					sps.event <- peripherals.Centre
				}

			case <-buttonPress:
				sps.event <- peripherals.Fire
			case <-buttonRelease:
				sps.event <- peripherals.NoFire

			case <-selectPress:
				sps.event <- peripherals.PanelSelectPress
			case <-selectRelease:
				sps.event <- peripherals.PanelSelectRelease

			case <-resetPress:
				sps.event <- peripherals.PanelResetPress
			case <-resetRelease:
				sps.event <- peripherals.PanelResetRelease
			}
		}
	}()

	return sps, nil
}

// GetInput implements the Controller interface
func (sps *SplaceStick) GetInput(_ string) (peripherals.Event, error) {
	select {
	case ev := <-sps.event:
		return ev, nil
	default:
		return peripherals.NoEvent, nil
	}
}
