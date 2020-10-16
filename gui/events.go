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

package gui

// Event represents all the different type of events that can occur in the gui
//
// Events are the things that happen in the gui, as a result of user interaction,
// and sent over a registered event channel.
type Event interface{}

// EventQuit is sent when the gui window is closed.
type EventQuit struct{}

// KeyMod identifies.
type KeyMod int

// list of valud key modifiers.
const (
	KeyModNone KeyMod = iota
	KeyModShift
	KeyModCtrl
	KeyModAlt
)

// EventKeyboard is the data that accompanies EventKeyboard events.
type EventKeyboard struct {
	Key  string
	Down bool
	Mod  KeyMod
}

// EventMouseMotion is the data that accompanies MouseEventMove events.
type EventMouseMotion struct {
	// as a fraction of the window's dimensions
	X float32
	Y float32
}

// MouseButton identifies the mouse button.
type MouseButton int

// list of valid MouseButtonIDs.
const (
	MouseButtonNone MouseButton = iota
	MouseButtonLeft
	MouseButtonRight
	MouseButtonMiddle
)

// EventMouseButton is the data that accompanies MouseEventMove events.
type EventMouseButton struct {
	Button MouseButton
	Down   bool
}

// EventDbgMouseButton is the data that accompanies MouseEventMove events.
type EventDbgMouseButton struct {
	Button   MouseButton
	Down     bool
	X        int
	Y        int
	HorizPos int
	Scanline int
}
