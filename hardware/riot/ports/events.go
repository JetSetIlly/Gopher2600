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

package ports

// Event represents the actions that can be performed at one of the VCS ports,
// either the panel or one of the two player ports.
type Event string

// List of defined events.
const (
	NoEvent Event = "NoEvent" // nil

	// joystick.
	Fire  Event = "Fire"  // bool
	Up    Event = "Up"    // bool
	Down  Event = "Down"  // bool
	Left  Event = "Left"  // bool
	Right Event = "Right" // bool

	// paddles.
	PaddleFire Event = "PaddleFire" // bool
	PaddleSet  Event = "PaddleSet"  // float64

	// keyboard.
	KeyboardDown Event = "KeyboardDown" // rune
	KeyboardUp   Event = "KeyboardUp"   // nil

	// panel.
	PanelSelect Event = "PanelSelect" // bool
	PanelReset  Event = "PanelReset"  // bool

	PanelSetColor      Event = "PanelSetColor"      // bool
	PanelSetPlayer0Pro Event = "PanelSetPlayer0Pro" // bool
	PanelSetPlayer1Pro Event = "PanelSetPlayer1Pro" // bool

	PanelToggleColor      Event = "PanelToggleColor"      // nil
	PanelTogglePlayer0Pro Event = "PanelTogglePlayer0Pro" // nil
	PanelTogglePlayer1Pro Event = "PanelTogglePlayer1Pro" // nil

	PanelPowerOff Event = "PanelPowerOff" // nil
)

// EventData is the value associated with the event. The underlying type should
// be restricted to bool, float32, or int. string is also acceptable but for
// simplicity of playback parsers, the strings "true" or "false" should not be
// used and numbers should be represented by float32 or int never as a string.
type EventData interface{}

// Playback implementations feed controller Events to the device on request
// with the CheckInput() function.
//
// Intended for playback of controller events previously recorded to a file on
// disk but usable for many purposes I suspect. For example, AI control.
type EventPlayback interface {
	// note the type restrictions on EventData in the type definition's
	// commentary
	GetPlayback() (PortID, Event, EventData, error)
}

// EventRecorder implementations mirror an incoming event.
//
// Implementations should be able to handle being attached to more than one
// peripheral at once. The ID parameter of the EventRecord() function will help
// to differentiate between multiple devices.
type EventRecorder interface {
	RecordEvent(PortID, Event, EventData) error
}
