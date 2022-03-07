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

// Package atarivoxengines contains implementations of the AtariVoxEngine
// interface, for use with the AtariVox peripheral.
package atarivoxengines

// AtariVoxEngine defines the operations required by any process that can
// interperet SpeakJet codes.
type AtariVoxEngine interface {
	// Quit instructs the engine to cleanup and quit
	Quit()

	// Interpret SpeakJet code and forward to engine
	SpeakJet(uint8)

	// Flush any outstanding instructions from previous calls to SpeakJet()
	Flush()
}
