// Panel uses the concurrent chip bus interface

package peripherals

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
)

// Panel represents the console's front control panel
type Panel struct {
	peripheral

	id PeriphID

	riot          memory.PeriphBus
	p0pro         bool
	p1pro         bool
	color         bool
	selectPressed bool
	resetPressed  bool
}

// NewPanel is the preferred method of initialisation for the Panel type
func NewPanel(riot memory.PeriphBus) *Panel {
	pan := &Panel{
		id:    PanelID,
		riot:  riot,
		color: true}

	pan.peripheral = peripheral{
		id:     pan.id,
		handle: pan.Handle}

	pan.commit()

	return pan
}

func (pan *Panel) commit() {
	// commit changes to RIOT memory
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

	if !pan.selectPressed {
		strobe |= 0x02
	}

	if !pan.resetPressed {
		strobe |= 0x01
	}

	pan.riot.PeriphWrite(vcssymbols.SWCHB, strobe)
}

// Handle interprets an event into the correct sequence of memory addressing
func (pan *Panel) Handle(event Event) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case PanelSelectPress:
		pan.selectPressed = true
	case PanelSelectRelease:
		pan.selectPressed = false
	case PanelResetPress:
		pan.resetPressed = true
	case PanelResetRelease:
		pan.resetPressed = false
	case PanelToggleColor:
		pan.color = !pan.color
	case PanelTogglePlayer0Pro:
		pan.p0pro = !pan.p0pro
	case PanelTogglePlayer1Pro:
		pan.p1pro = !pan.p1pro
	case PanelPowerOff:
		return errors.NewFormattedError(errors.PowerOff)
	default:
		return errors.NewFormattedError(errors.UnknownPeriphEvent, pan.id, event)
	}

	pan.commit()

	// record event with the transcriber
	if pan.scribe != nil {
		return pan.scribe.Transcribe(pan.id, event)
	}

	return nil
}
