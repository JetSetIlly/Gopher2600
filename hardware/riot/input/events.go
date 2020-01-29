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
type Event string

// List of defined events
const (
	NoEvent Event = "NoEvent"

	// the controller has been unplugged
	Unplug Event = "Unplug"

	// joystick
	Fire  Event = "Fire"  // bool
	Up    Event = "Up"    // bool
	Down  Event = "Down"  // bool
	Left  Event = "Left"  // bool
	Right Event = "Right" // bool

	// panel
	PanelSelect Event = "PanelSelect" // bool
	PanelReset  Event = "PanelReset"  // bool

	PanelSetColor      Event = "PanelSetColor"      // bool
	PanelSetPlayer0Pro Event = "PanelSetPlayer0Pro" // bool
	PanelSetPlayer1Pro Event = "PanelSetPlayer1Pro" // bool

	PanelToggleColor      Event = "PanelToggleColor"      // nil
	PanelTogglePlayer0Pro Event = "PanelTogglePlayer0Pro" // nil
	PanelTogglePlayer1Pro Event = "PanelTogglePlayer1Pro" // nil

	// paddles
	PaddleFire Event = "PaddleFire" // bool
	PaddleSet  Event = "PaddleSet"  // float64

	// keyboard (only need down event)
	KeyboardDown Event = "KeyboardDown" // rune
	KeyboardUp   Event = "KeyboardUp"   // nil

	PanelPowerOff Event = "PanelPowerOff"
)

// EventValue is the value associated with the event. The underlying type
// should be restricted to bool, float32, or int. string is also acceptable but
// for simplicity of playback parsers, "true" or "false" should not be used and
// numbers should be represented by float32 or int.
type EventValue interface{}
