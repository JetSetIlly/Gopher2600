package peripherals

// Event represents the possible actions that can be performed with a
// controller
type Event int

// list of defined Events
const (
	NoEvent Event = iota

	// the controller has been unplugged
	Unplugged

	// joystick
	Fire
	NoFire
	Up
	NoUp
	Down
	NoDown
	Left
	NoLeft
	Right
	NoRight

	// TODO: paddle and keyboard controllers

	// for convenience, a controller implementation can interact with the panel
	PanelSelectPress
	PanelSelectRelease
	PanelResetPress
	PanelResetRelease
	PanelToggleColor
	PanelTogglePlayer0Pro
	PanelTogglePlayer1Pro

	// PanelPowerOff is a special event and should probably be handled outside
	// of the panel implementation
	PanelPowerOff
)
