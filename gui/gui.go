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
// Note that many contexts where GUI is used also expect the GUI instance to
// implement the Television interface from the television package. This is
// probably best achieved by embedding an actual television implementation.
//
// In practice, the GUI instance may also implement the Renderer and AudioMixer
// interfaces, also from the television package.
type GUI interface {
	// All GUIs should implement a MetaPixelRenderer even if only a stub
	MetaPixelRenderer

	// returns true if GUI is currently visible. false if not
	IsVisible() bool

	// send a request to set a gui feature
	SetFeature(request FeatureReq, args ...interface{}) error

	// the event channel is used to by the GUI implementation to send
	// information back to the main program. the GUI may or may not be in its
	// own go routine but regardless, the event channel is used for this
	// purpose.
	SetEventChannel(chan Event)

	// Service() should not pause or loop longer than necessary (if at all). It
	// MUST ONLY by called as part of a larger loop from the main thread. It
	// should service all gui events that are not safe to do in sub-threads.
	//
	// If the GUI framework does not require this sort of thread safety then
	// there is no need for the Service() function to do anything.
	Service()
}
