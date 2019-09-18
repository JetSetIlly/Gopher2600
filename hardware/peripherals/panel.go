// Panel uses the concurrent chip bus interface

package peripherals

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
	"strings"
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

// MachineInfoTerse returns the panel information in terse format
func (pan *Panel) MachineInfoTerse() string {
	return pan.MachineInfo()
}

// MachineInfo returns the panel information in verbose format
func (pan *Panel) MachineInfo() string {
	s := strings.Builder{}

	s.WriteString("p0=")
	if pan.p0pro {
		s.WriteString("pro")
	} else {
		s.WriteString("am")
	}

	s.WriteString(", p1=")
	if pan.p1pro {
		s.WriteString("pro")
	} else {
		s.WriteString("am")
	}

	s.WriteString(", ")

	if pan.color {
		s.WriteString("col")
	} else {
		s.WriteString("b&w")
	}

	return s.String()
}

// map String to MachineInfo
func (pan *Panel) String() string {
	return pan.MachineInfo()
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

	pan.riot.PeriphWrite(addresses.SWCHB, strobe, 0xff)
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
	case PanelSetColor:
		pan.color = true
	case PanelSetBlackAndWhite:
		pan.color = false
	case PanelSetPlayer0Am:
		pan.p0pro = false
	case PanelSetPlayer1Am:
		pan.p1pro = false
	case PanelSetPlayer0Pro:
		pan.p0pro = true
	case PanelSetPlayer1Pro:
		pan.p1pro = true
	case PanelPowerOff:
		return errors.New(errors.PowerOff)
	default:
		return errors.New(errors.UnknownPeriphEvent, pan.id, event)
	}

	pan.commit()

	// record event with the transcriber
	if pan.scribe != nil {
		return pan.scribe.Transcribe(pan.id, event)
	}

	return nil
}
