package input

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
	"strings"
)

// Panel represents the console's front control panel
type Panel struct {
	device

	riot          bus.InputDeviceBus
	p0pro         bool
	p1pro         bool
	color         bool
	selectPressed bool
	resetPressed  bool
}

// NewPanel is the preferred method of initialisation for the Panel type
func NewPanel(riot bus.InputDeviceBus) *Panel {
	pan := &Panel{
		riot:  riot,
		color: true}

	pan.device = device{
		id:     PanelID,
		handle: pan.Handle}

	pan.write()

	return pan
}

func (pan *Panel) String() string {
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

func (pan *Panel) write() {
	// commit changes to RIOT memory
	v := uint8(0)

	// pins 2, 4 and 5 are not used and always value value of 1
	v |= 0x20
	v |= 0x10
	v |= 0x04

	if pan.p0pro {
		v |= 0x80
	}

	if pan.p1pro {
		v |= 0x40
	}

	if pan.color {
		v |= 0x08
	}

	if !pan.selectPressed {
		v |= 0x02
	}

	if !pan.resetPressed {
		v |= 0x01
	}

	pan.riot.InputDeviceWrite(addresses.SWCHB, v, 0xff)
}

// Handle interprets an event into the correct sequence of memory addressing
func (pan *Panel) Handle(event Event) error {
	switch event {
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
	case NoEvent:
		return nil
	default:
		return errors.New(errors.UnknownInputEvent, pan.id, event)
	}

	pan.write()

	// record event with the EventRecorder
	if pan.recorder != nil {
		return pan.recorder.RecordEvent(pan.id, event)
	}

	return nil
}
