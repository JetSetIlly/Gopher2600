// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

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

	// panel
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
