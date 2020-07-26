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

package input

// Playback implementations feed controller Events to the device on request
// with the CheckInput() function.
//
// Intended for playback of controller events previously recorded to a file on
// disk but usable for many purposes I suspect. For example, AI control.
type Playback interface {
	// note the type restrictions on EventData in the type definition's
	// commentary
	CheckInput(id ID) (Event, EventData, error)
}

// EventRecorder implementations mirror an incoming event.
//
// Implementations should be able to handle being attached to more than one
// peripheral at once. The ID parameter of the EventRecord() function will help
// to differentiate between multiple devices.
type EventRecorder interface {
	RecordEvent(ID, Event, EventData) error
}
