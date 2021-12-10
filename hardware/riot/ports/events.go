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

import (
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

// Event represents the actions that can be performed at one of the VCS ports,
// either the panel or one of the two player ports.
type Event string

// List of defined events. The comment indicates the expected type of the
// associated EventData. In all cases the EventData can also be
// EventDataPlayback, indicating that the event has been read from a playback
// file and will need further parsing.
const (
	NoEvent Event = "NoEvent" // nil

	// fire will be interpreted by both stick and paddle depending on which
	// controller is attached.
	Fire Event = "Fire" // bool

	// joystick.
	Centre    Event = "Centre"    // nil
	Up        Event = "Up"        // EventDataStick
	Down      Event = "Down"      // EventDataStick
	Left      Event = "Left"      // EventDataStick
	Right     Event = "Right"     // EventDataStick
	LeftUp    Event = "LeftUp"    // EventDataStick
	LeftDown  Event = "LeftDown"  // EventDataStick
	RightUp   Event = "RightUp"   // EventDataStick
	RightDown Event = "RightDown" // EventDataStick

	// paddles.
	PaddleSet Event = "PaddleSet" // float32

	// keyboard.
	KeypadDown Event = "KeypadDown" // rune
	KeypadUp   Event = "KeypadUp"   // nil

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

// Sentinal error returned when PanelPowerOff event is received.
const (
	PowerOff = "emulated machine has been powered off"
)

// EventData is the value associated with the event. The underlying type should
// be restricted to bool, float32, or int. string is also acceptable but for
// simplicity of playback parsers, the strings "true" or "false" should not be
// used and numbers should be represented by float32 or int never as a string.
type EventData interface{}

// EventData is from playback file and will need further parsing.
type EventDataPlayback string

// Event data for stick types.
type EventDataStick string

// List of valid values for EventDataStick.
//
// A note on the values. DataStickTrue will set the bits associated with the
// Event and DataStickFalse will unset the bits. DataStickSet will set the bits
// AND unset any bits not associated the events.
//
// When you use DataStickSet and when you use DataStickTrue/DataStickFalse
// depends on the harware controller being used to create the input. For DPad
// devices the DataStickSet should be used; for keyboard devices then the
// true/false forms are preferred.
//
// NB: true and false are used to maintain compatibility with earlier version
// of the playback fileformat.
const (
	DataStickTrue  EventDataStick = "true"
	DataStickFalse EventDataStick = "false"
	DataStickSet   EventDataStick = "set"
)

// InputEvent defines the data required for single input event.
type InputEvent struct {
	Port plugging.PortID
	Ev   Event
	D    EventData
}

// TimedInputEvent embeds the InputEvent type and adds a Time field (time
// measured by TelevisionCoords).
type TimedInputEvent struct {
	Time coords.TelevisionCoords
	InputEvent
}
