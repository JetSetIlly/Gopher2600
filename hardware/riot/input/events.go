package input

// Event represents the possible actions that can be performed by the user
// when interacting with the console
type Event int

// List of defined events. Do not monkey with the ordering of these
// constants unless you know what you're doing. Existing playback scripts will
// probably break.
const (
	NoEvent Event = iota

	// the controller has been unplugged
	Unplug

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

	// for convenience, a controller implementation can interact with the panel
	PanelSelectPress
	PanelSelectRelease
	PanelResetPress
	PanelResetRelease
	PanelToggleColor
	PanelTogglePlayer0Pro
	PanelTogglePlayer1Pro
	PanelSetColor
	PanelSetBlackAndWhite
	PanelSetPlayer0Am
	PanelSetPlayer1Am
	PanelSetPlayer0Pro
	PanelSetPlayer1Pro

	// !!TODO: paddle and keyboard controllers

	PanelPowerOff Event = 255
)
