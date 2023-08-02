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

// Pacakge input coordinates all the different types of input into the VCS.
// The types of input handled by the package include:
//
// 1) Immediate input from the user (see userinput package)
// 2) Playback for a previously recorded script (see recorder package)
// 3) Driven events from a driver emulation (see below))
// 4) Pushed events
//
// In addition it also coordinates the passing of input to other packages that
// need to know about an input event.
//
// 1) The RIOT ports (see Ports package)
// 2) Events to be recorded to a playback script (see recorder package)
// 3) Events to be driven to a passenger emulation  (see below)
//
// The input package will handle impossible situations are return an error when
// appropriate. For example, it is not possible for an emulation to be a
// playback and a recorder at the same time.
//
// Points 1 in both lists is the normal type of input you would expect in an
// emultor that allows people to play games and as such, won't be discussed
// further.
//
// Points 2 in the lists is well covered by the recorder package.
//
// Points 3 in both lists above refer to driven events and driver & passenger
// emulations. This system is a way os synchronising two emulations such that
// the input of one drives the input of the other. The input package ensures
// that input events occur in bothe emulations at the same time - time being
// measure by the coordinates of the TV instances attached to the emulations.
//
// For an example of a driven emulation see the comparison package.
//
// Lastly, pushed events, as refered to in point 4, are events that have
// arrived from a different goroutine. For an example of pushed events see the
// various bot implementations in the bots package.
package input
