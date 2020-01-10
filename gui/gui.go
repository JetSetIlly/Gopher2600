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

// GUI defines the operations that can be performed on visual user interfaces.
//
// Currently, GUI implementations expect also to be an instance of
// television.Television. This way a single object can be used in both GUI and
// television contexts. In practice, the GUI instance may also implement the
// Renderer and AudioMixer interfaces from the television packages but this is
// not mandated by the GUI interface.
type GUI interface {
	// All GUIs should implement a MetaPixelRenderer even if only a stub
	MetaPixelRenderer

	// returns true if GUI is currently visible. false if not
	IsVisible() bool

	// send a request to set a gui feature
	SetFeature(request FeatureReq, args ...interface{}) error

	// the event channel is used to by the GUI implementation to send
	// information back to the main program. the GUI may or may not be in its
	// own go routine but in regardless, the event channel is used for this
	// purpose.
	SetEventChannel(chan (Event))
}

// EventsLoop defines the service events function required by a GUI. It is
// sometimes necessary to service events in a different goroutine to where the
// gui was created. In those cases the Events interface is more convenient to
// move around.
type EventsLoop interface {
	// ServiceEvents should not pause or loop longer than necessary (if at
	// all). An EventsLoop interface will be called as part of a larger loop
	// and will be called as often as required.
	ServiceEvents() bool
}
