package peripherals

import (
	"gopher2600/errors"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
)

// Ports is the containing structure for the two player ports
type Ports struct {
	Player0 *player
	Player1 *player
}

// NewPorts is the preferred method of initialisation for the Ports type
func NewPorts(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *Ports {
	pt := new(Ports)
	pt.Player0 = newPlayer0(riot, tia, panel)
	pt.Player1 = newPlayer1(riot, tia, panel)
	return pt
}

// A player instance is used by controllers to communicate with the VCS
type player struct {
	peripheral

	id PeriphID

	riot  memory.PeriphBus
	tia   memory.PeriphBus
	panel *Panel

	// joystick
	//	o stickAddr is in the RIOT area of memory
	stickAddr uint16 // RIOT address
	//  o stickMask indicates which bits in stickValue are relevant
	stickMask uint8
	//  o stickValue is sent to the RIOT address where it is masked and written
	//		apporpriately
	stickValue uint8
	//	o poth player ports write to the same joystick address but in a
	//		slightly different way. the stickFunc allows an easy way to
	//		transform player0 values to player1 values
	stickFunc func(uint8) uint8

	// joystick fire button
	buttonAddr uint16 // TIA address
	buttonMask uint8
}

func newPlayer0(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *player {
	pl := &player{
		id:    PlayerZeroID,
		riot:  riot,
		tia:   tia,
		panel: panel,

		stickAddr:  addresses.SWCHA,
		stickMask:  0xf0,
		stickValue: 0xf0,
		stickFunc:  func(n uint8) uint8 { return n },

		buttonAddr: addresses.INPT4,
		buttonMask: 0xff,
	}

	pl.peripheral = peripheral{
		id:     pl.id,
		handle: pl.Handle}

	return pl
}

func newPlayer1(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *player {
	pl := &player{
		id:    PlayerOneID,
		riot:  riot,
		tia:   tia,
		panel: panel,

		stickAddr:  addresses.SWCHA,
		stickMask:  0x0f,
		stickValue: 0xf0,
		stickFunc:  func(n uint8) uint8 { return n << 4 },

		buttonAddr: addresses.INPT5,
		buttonMask: 0xff,
	}

	pl.peripheral = peripheral{
		id:     pl.id,
		handle: pl.Handle}

	return pl
}

// Handle interprets an event into the correct sequence of memory addressing
func (pl *player) Handle(event Event) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case Left:
		pl.stickValue ^= 0x4f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Right:
		pl.stickValue ^= 0x8f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Up:
		pl.stickValue ^= 0x1f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Down:
		pl.stickValue ^= 0x2f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoLeft:
		pl.stickValue |= 0x4f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoRight:
		pl.stickValue |= 0x8f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoUp:
		pl.stickValue |= 0x1f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case NoDown:
		pl.stickValue |= 0x2f
		pl.riot.PeriphWrite(pl.stickAddr, pl.stickFunc(pl.stickValue), pl.stickMask)
	case Fire:
		pl.tia.PeriphWrite(pl.buttonAddr, 0x00, pl.buttonMask)
	case NoFire:
		pl.tia.PeriphWrite(pl.buttonAddr, 0x80, pl.buttonMask)

	// for convenience, a controller implementation can interact with the panel
	// note that the function returns the result of panel.Handle straightaway
	// and will cause a transcriber to miss the event (the event may be written by
	// a transcriber attached to the panel)
	case PanelSelectPress:
		return pl.panel.Handle(PanelSelectPress)
	case PanelSelectRelease:
		return pl.panel.Handle(PanelSelectPress)
	case PanelResetPress:
		return pl.panel.Handle(PanelResetPress)
	case PanelResetRelease:
		return pl.panel.Handle(PanelResetRelease)

	case Unplugged:
		return errors.NewFormattedError(errors.PeriphUnplugged, pl.id)

	// return now if there is no event to process
	default:
		return errors.NewFormattedError(errors.UnknownPeriphEvent, pl.id, event)
	}

	// record event with the transcriber
	if pl.scribe != nil {
		return pl.scribe.Transcribe(pl.id, event)
	}

	return nil
}
