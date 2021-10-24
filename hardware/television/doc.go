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

// Package television implements the output device of the emulated VCS.
//
// It is possible for instances of television to be embedded in other type
// structure, thereby extending the "features" of the television and allowing
// the extended type to be used wherever the Television interface is required.
// The digest package is a good example of this idea.
//
// It is important to note that the television package does not render pixels
// or mix sound. Instead, the television interface exposes two functions,
// AddPixelRenderer() and AddAudioMixer(). These can be used to add as many
// renderers and mixers as required.
//
// For audio outputs that are time sensitive, the AddRealtimeAudioMixer()
// function should be used. This requires a slightly different implementation
// (see RealtimeAudioMixer interface) and only one such implemenation can be
// registered at any one time.
//
// There is also the FrameTrigger and PauseTrigger interfaces for applications
// that have limited need for a full pixel renderer.
//
// The main means of communication is the Signal() function. This function
// accepts an instance of SignalAttributes which gives details of how the
// television should be behaving.
//
// Note that the television implementation no longer attempts to report the
// same frame/scanline/clock information as Stella. Early versions of the
// implementation did because it facilitated A/B testing but since we're
// now confident that the TIA emulation is correct the need to keep in "sync"
// with Stella is no longer required.
//
// The reference implementation also handles framerate limiting according to
// the current incoming TV signal. For debugging purposes, the framerate can
// also be set to a specific value
//
// Framesize adaptation is also handled by the television package. Current
// information about the frame can be acquired with GetFrameInfo(). FrameInfo
// will also be sent to the PixelRenderers as appropriate.
//
// Screen Rolling
//
// Screen rolling is not handled by the television package. However, the synced
// argument of the NewFrame() function in the PixelRenderer and FrameTrigger
// interfaces can be used to implement it if required. Something like this:
//
//    1) If Synced is false, note scanline of last plot (unsynedScanline)
//    2) For every SetPixel() add unsyncedScanline to the Scanline value in
//       the SignalAttributes struct (adjustedScanline)
//    3) Bring adjustedScanline into range by modulo ScanlinesTotal of the
//       current TV specification.
//
// Recovery from a screen roll should also be emulated. A good way of doing
// this is to reduce unsyncedScanline by a percentage (80% say) on synced
// frames (or every other synced frame) after an unsynced frame.
//
// A good additionl policy would be to only roll if several, consecutive
// unsynced frames are indicated.
package television
