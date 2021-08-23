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

package mapper

// Event defines specific and discrete events that might occur in a cartridge
// mapper. The intention is that these are defined for events that are not
// "normal". For example, bank switching is so common and so frequent as to be
// considered normal. Tape activity for the supercharger on the other hand is
// not normal and needs special handling from the core emulation.
type Event int

// List of currently defined activities.
const (
	// LoadStarted is raised for Supercharger mapper whenever a new tape read
	// sequence if started
	EventSuperchargerLoadStarted Event = iota

	// If Supercharger is loading from a fastload binary then this event is
	// raised when the loading has been completed
	EventSuperchargerFastloadEnded

	// If Supercharger is loading from a sound file (eg. mp3 file) then these
	// events area raised when the loading has started/completed
	EventSuperchargerSoundloadStarted
	EventSuperchargerSoundloadEnded

	// tape is rewinding
	EventSuperchargerSoundloadRewind

	// PlusROM cartridge has been inserted
	EventPlusROMInserted

	// PlusROM network activity
	EventPlusROMNetwork
)
