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

package gui

// Events are the things that happen in the gui, as a result of user interaction,
// and sent over a registered event channel.

// EventID idintifies the type of event taking place
type EventID int

// list of valid events
const (
	EventWindowClose EventID = iota
	EventKeyboard
	EventMouseLeft
	EventMouseRight
)

// KeyMod identifies
type KeyMod int

// list of valud key modifiers
const (
	KeyModNone KeyMod = iota
	KeyModShift
	KeyModCtrl
	KeyModAlt
)

// EventData represents the data that is associated with an event
type EventData interface{}

// Event is the structure that is passed over the event channel
//
// Do not confuse this with the peripheral Event type.
type Event struct {
	ID   EventID
	Data EventData
}

// EventDataKeyboard is the data that accompanies EvenKeyboard events
type EventDataKeyboard struct {
	Key  string
	Down bool
	Mod  KeyMod
}

// EventDataMouse is the data that accompanies EventMouse events
type EventDataMouse struct {
	Down     bool
	X        int
	Y        int
	HorizPos int
	Scanline int
}
