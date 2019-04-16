package peripherals

// Event represents the possible actions that can be performed with a
// controller
type Event int

// list of defined Events
// TODO: paddle and keyboard events
const (
	NoEvent Event = iota

	// joystick
	Up
	Down
	Left
	Right
	Centre
	Fire
	NoFire

	// TODO: paddle and keyboard controllers

	// for convenience, a controller implementation can interact with the panel
	PanelSelectPress
	PanelResetPress
	PanelSelectRelease
	PanelResetRelease
)

// Controller defines the operations required for VCS controllers
type Controller interface {
	GetInput() Event
}
