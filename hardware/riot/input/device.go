package input

import "gopher2600/errors"

// ID differentiates the different devices attached to the console. note that
// PlayerZero and PlayerOne can have different types of devices attached to
// them (joysticks, paddles, keyboards)
type ID int

// List of defined IDs
const (
	PlayerZeroID ID = iota
	PlayerOneID
	PanelID
	NumIDs
)

type device struct {
	id     ID
	handle func(Event) error

	controller     Controller
	prevController Controller
	recorder       EventRecorder
}

// Attach registers a controller implementation with the panel
func (dev *device) Attach(controller Controller) {
	if controller == nil {
		dev.controller = dev.prevController
		dev.prevController = nil
	} else {
		dev.prevController = dev.controller
		dev.controller = controller
	}
}

// AttachEventRecorder registers the presence of a transcriber implementation.
// use an argument of nil to disconnect an existing scribe
func (dev *device) AttachEventRecorder(scribe EventRecorder) {
	dev.recorder = scribe
}

// CheckInput makes sure attached controllers have submitted their latest input
func (dev *device) CheckInput() error {
	if dev.controller != nil {
		ev, err := dev.controller.CheckInput(dev.id)
		if err != nil {
			return err
		}

		err = dev.handle(ev)
		if err != nil {
			if !errors.Is(err, errors.InputDeviceUnplugged) {
				return err
			}
			dev.Attach(nil)
		}
	}

	return nil
}
