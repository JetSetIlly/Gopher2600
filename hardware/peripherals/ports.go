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

	lastJoystickValue uint8
}

// NewPorts is the preferred method of initialisation for the Ports type
func NewPorts(riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *Ports {
	pt := new(Ports)
	pt.Player0 = newPlayer0(pt, riot, tia, panel)
	pt.Player1 = newPlayer1(pt, riot, tia, panel)
	pt.lastJoystickValue = 0xff
	return pt
}

// A player instance is used by controllers to communicate with the VCS
type player struct {
	peripheral

	// pointer back to the containing Ports stuct. we need this because both
	// player instances need to know what lastJoystickValue is
	ports *Ports

	id PeriphID

	riot  memory.PeriphBus
	tia   memory.PeriphBus
	panel *Panel

	// joysticks
	joystick   uint16 // RIOT address
	fireButton uint16 // TIA address

	// currentState uses the most significant nibble
	currentState uint8

	// poth player ports write to the same joystick address but in a slightly
	// different way. the joystickFunc allows an easy way to transform player0
	// values to player1 values
	joystickFunc func(uint8) uint8
}

func newPlayer0(pt *Ports, riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *player {
	pl := &player{
		id:           PlayerOneID,
		ports:        pt,
		riot:         riot,
		tia:          tia,
		panel:        panel,
		joystick:     addresses.SWCHA,
		fireButton:   addresses.INPT4,
		currentState: 0xf0,
		joystickFunc: func(n uint8) uint8 { return n }}

	pl.peripheral = peripheral{
		id:     pl.id,
		handle: pl.Handle}

	pl.ports.lastJoystickValue |= pl.joystickFunc(pl.currentState)
	pl.ports.lastJoystickValue &= pl.joystickFunc(0xff)
	pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	pl.tia.PeriphWrite(pl.fireButton, 0x80)

	return pl
}

func newPlayer1(pt *Ports, riot memory.PeriphBus, tia memory.PeriphBus, panel *Panel) *player {
	pl := &player{
		id:           PlayerTwoID,
		ports:        pt,
		riot:         riot,
		tia:          tia,
		panel:        panel,
		joystick:     addresses.SWCHA,
		fireButton:   addresses.INPT5,
		currentState: 0xf0,
		joystickFunc: func(n uint8) uint8 { return (n >> 4) | (n << 4) }}

	pl.peripheral = peripheral{
		id:     pl.id,
		handle: pl.Handle}

	pl.ports.lastJoystickValue |= pl.joystickFunc(pl.currentState)
	pl.ports.lastJoystickValue &= pl.joystickFunc(0xff)
	pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	pl.tia.PeriphWrite(pl.fireButton, 0x80)

	return pl
}

// Handle interprets an event into the correct sequence of memory addressing
func (pl *player) Handle(event Event) error {
	switch event {

	// do nothing at all if event is a NoEvent
	case NoEvent:
		return nil

	case Left:
		pl.currentState ^= 0x4f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case Right:
		pl.currentState ^= 0x8f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case Up:
		pl.currentState ^= 0x1f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case Down:
		pl.currentState ^= 0x2f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case NoLeft:
		pl.currentState |= 0x4f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case NoRight:
		pl.currentState |= 0x8f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case NoUp:
		pl.currentState |= 0x1f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case NoDown:
		pl.currentState |= 0x2f
		pl.ports.lastJoystickValue |= pl.joystickFunc(0xf0)
		pl.ports.lastJoystickValue &= pl.joystickFunc(pl.currentState)
		pl.riot.PeriphWrite(pl.joystick, pl.ports.lastJoystickValue)
	case Fire:
		pl.tia.PeriphWrite(pl.fireButton, 0x00)
	case NoFire:
		pl.tia.PeriphWrite(pl.fireButton, 0x80)

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
