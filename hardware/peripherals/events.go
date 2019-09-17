package peripherals

// Event represents the possible actions that can be performed with a
// controller
type Event int

// list of defined events
//
// *** do not monkey with the ordering of these constants unless you know what
// you're doing. existing playback scripts will probably break ***
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
