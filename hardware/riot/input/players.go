package input

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/bus"
)

// Player type is the underlying implementation for each emulated player port
// on the VCS
type Player struct {
	device

	riot bus.InputDeviceBus
	tia  bus.InputDeviceBus

	// address in RIOT memory for joystick direction input
	stickAddr uint16

	// value indicating joystick state
	stickValue uint8

	// player port 0 and 1 write the stickValue to different nibbles of the
	// stickAddr. stickFunc allows us to transform that value with the help of
	// stickMask
	stickFunc func(uint8) uint8
	stickMask uint8

	// the address in TIA memory for joystick fire button
	buttonAddr uint16
}

// NewPlayer0 creates a new instance of the player type for the player 1 port
func NewPlayer0(riot bus.InputDeviceBus, tia bus.InputDeviceBus) *Player {
	pl := &Player{
		riot: riot,
		tia:  tia,

		stickAddr:  addresses.SWCHA,
		stickMask:  0xf0,
		stickValue: 0xf0,
		stickFunc:  func(n uint8) uint8 { return n },

		buttonAddr: addresses.INPT4,
	}

	pl.device = device{
		id:     PlayerZeroID,
		handle: pl.Handle}

	return pl
}

// NewPlayer1 creates a new instance of the player type for the player 1 port
func NewPlayer1(riot bus.InputDeviceBus, tia bus.InputDeviceBus) *Player {
	pl := &Player{
		riot: riot,
		tia:  tia,

		stickAddr:  addresses.SWCHA,
		stickMask:  0x0f,
		stickValue: 0xf0,
		stickFunc:  func(n uint8) uint8 { return n << 4 },

		buttonAddr: addresses.INPT5,
	}

	pl.device = device{
		id:     PlayerOneID,
		handle: pl.Handle}

	return pl
}

// Handle interprets an event into the correct sequence of memory addressing
func (pl *Player) Handle(event Event) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case Left:
		pl.stickValue ^= 0x4f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Right:
		pl.stickValue ^= 0x8f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Up:
		pl.stickValue ^= 0x1f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Down:
		pl.stickValue ^= 0x2f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoLeft:
		pl.stickValue |= 0x4f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoRight:
		pl.stickValue |= 0x8f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoUp:
		pl.stickValue |= 0x1f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoDown:
		pl.stickValue |= 0x2f
		pl.riot.InputDeviceWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Fire:
		pl.tia.InputDeviceWrite(pl.buttonAddr, 0x00, 0xff)
	case NoFire:
		pl.tia.InputDeviceWrite(pl.buttonAddr, 0x80, 0xff)

	case Unplug:
		return errors.New(errors.InputDeviceUnplugged, pl.id)

	// return now if there is no event to process
	default:
		return errors.New(errors.UnknownInputEvent, pl.id, event)
	}

	// record event with the EventRecorder
	if pl.recorder != nil {
		return pl.recorder.RecordEvent(pl.id, event)
	}

	return nil
}
