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
	"errors"
	"fmt"
	"strconv"
	"strings"

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

	// fire is the standard fire button present on all controller types
	Fire Event = "Fire" // bool

	// second button is treated as the paired paddle fire button as well as the
	// B button on a game pad
	SecondFire Event = "SecondFire" // bool

	// joystick
	Centre    Event = "Centre"    // nil
	Up        Event = "Up"        // EventDataStick
	Down      Event = "Down"      // EventDataStick
	Left      Event = "Left"      // EventDataStick
	Right     Event = "Right"     // EventDataStick
	LeftUp    Event = "LeftUp"    // EventDataStick
	LeftDown  Event = "LeftDown"  // EventDataStick
	RightUp   Event = "RightUp"   // EventDataStick
	RightDown Event = "RightDown" // EventDataStick

	// paddles
	PaddleSet Event = "PaddleSet" // EventDataPaddle

	// keyboard
	KeypadDown Event = "KeypadDown" // rune
	KeypadUp   Event = "KeypadUp"   // nil

	// panel
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

// sentinal error returned when PanelPowerOff event is received.
var PowerOff = errors.New("emulated machine has been powered off")

// EventData is the value associated with the event. The underlying type should
// be restricted to bool, float32, or int. string is also acceptable but for
// simplicity of playback parsers, the strings "true" or "false" should not be
// used and numbers should be represented by float32 or int never as a string.
type EventData interface{}

// EventData is from playback file and will need further parsing.
type EventDataPlayback string

// Event data for stick types.
type EventDataStick string

// Event data for paddle types. first entry is for primary paddle for the part
// (ie. INP0 or INP2) and the second entry is for the secondary paddle (ie.
// INP1 or INP3)
//
// The values are relative (to the current position) if the Relative field is
// set to true
//
// For non-relative values (ie. absolute values) the value should be scaled to
// be in the range of -32768 and +32767
type EventDataPaddle struct {
	A        int16
	B        int16
	Relative bool
}

// String implements the string.Stringer interface and is intended to be used
// when writing to a playback file
func (ev EventDataPaddle) String() string {
	return fmt.Sprintf("%d;%d;%v", ev.A, ev.B, ev.Relative)
}

// FromString is the inverse of the String() function
func (ev *EventDataPaddle) FromString(s string) error {
	sp := strings.Split(s, ";")

	if len(sp) != 3 {
		return fmt.Errorf("wrong number of values in paddle string")
	}

	f, err := strconv.ParseInt(sp[0], 10, 32)
	if err != nil {
		return fmt.Errorf("illegal value in paddle string")
	}
	ev.A = int16(f)

	f, err = strconv.ParseInt(sp[1], 10, 32)
	if err != nil {
		return fmt.Errorf("illegal value in paddle string")
	}
	ev.B = int16(f)

	switch sp[2] {
	case "true":
		ev.Relative = true
	case "false":
		ev.Relative = false
	default:
		return fmt.Errorf("illegal value in paddle string")
	}

	return nil
}

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
