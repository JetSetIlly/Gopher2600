package peripherals

import "gopher2600/errors"

// PeriphID differentiates peripherals attached to the console
type PeriphID int

// list of defined PeriphIDs
const (
	PlayerZeroID PeriphID = iota
	PlayerOneID
	PanelID
	NumPeriphIDs
)

type peripheral struct {
	id     PeriphID
	handle func(Event) error

	controller     Controller
	prevController Controller
	scribe         Transcriber
}

// Attach registers a controller implementation with the panel
func (prp *peripheral) Attach(controller Controller) {
	if controller == nil {
		prp.controller = prp.prevController
		prp.prevController = nil
	} else {
		prp.prevController = prp.controller
		prp.controller = controller
	}
}

// AttachTranscriber registers the presence of a transcriber implementation. use an
// argument of nil to disconnect an existing scribe
func (prp *peripheral) AttachTranscriber(scribe Transcriber) {
	prp.scribe = scribe
}

// Strobe makes sure the controller have submitted their latest input
func (prp *peripheral) Strobe() error {
	if prp.controller != nil {
		ev, err := prp.controller.GetInput(prp.id)
		if err != nil {
			return err
		}

		err = prp.handle(ev)
		if err != nil {
			switch err := err.(type) {
			case errors.FormattedError:
				if err.Errno != errors.PeriphUnplugged {
					return err
				}
				prp.Attach(nil)
			default:
				return err
			}
		}
	}

	return nil
}
