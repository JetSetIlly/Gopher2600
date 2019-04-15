package peripherals

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
	"strings"
)

// DigitalStick is the minimal implementation for the VCS joystick
type DigitalStick struct {
	riot         memory.PeriphBus
	tia          memory.PeriphBus
	stickAddress uint16
	fireAddress  uint16
	transcriber  Transcriber
}

// NewDigitalStick is the preferred method of initialisation the DigitalStick
// type
func NewDigitalStick(player int, riot memory.PeriphBus, tia memory.PeriphBus) (*DigitalStick, error) {
	dst := &DigitalStick{riot: riot, tia: tia}

	// TODO: make all this work with a second contoller. for now, initialise
	// and asssume that there is just one controller for player 0
	dst.riot.PeriphWrite(vcssymbols.SWCHA, 0xff)
	dst.tia.PeriphWrite(vcssymbols.INPT4, 0x80)
	dst.tia.PeriphWrite(vcssymbols.INPT5, 0x80)

	if player == 0 {
		dst.stickAddress = vcssymbols.SWCHA
		dst.fireAddress = vcssymbols.INPT4
	} else if player == 1 {
		dst.stickAddress = vcssymbols.SWCHB
		dst.fireAddress = vcssymbols.INPT5
	} else {
		return nil, errors.NewFormattedError(errors.NoPlayerPort)
	}

	return dst, nil
}

// Handle implements the Controller interface
func (dst DigitalStick) Handle(action string) error {
	switch strings.ToUpper(action) {
	case "LEFT":
		dst.riot.PeriphWrite(dst.stickAddress, 0xbf)
	case "RIGHT":
		dst.riot.PeriphWrite(dst.stickAddress, 0x7f)
	case "UP":
		dst.riot.PeriphWrite(dst.stickAddress, 0xef)
	case "DOWN":
		dst.riot.PeriphWrite(dst.stickAddress, 0xdf)
	case "CENTRE", "CENTER":
		dst.riot.PeriphWrite(dst.stickAddress, 0xff)
	case "FIRE":
		dst.tia.PeriphWrite(dst.fireAddress, 0x00)
	case "NOFIRE":
		dst.tia.PeriphWrite(dst.fireAddress, 0x80)
	}

	if dst.transcriber != nil {
		dst.transcriber.Transcribe(action)
	}

	return nil
}

// RegisterTranscriber implements the Controller interface
func (dst *DigitalStick) RegisterTranscriber(trans Transcriber) {
	dst.transcriber = trans
}

// Strobe implements the Controller interface
func (dst *DigitalStick) Strobe() error {
	return nil
}
