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
// It is important to note that the Television type does not render pixels or
// mix sound. Instead, the television interface exposes two functions,
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
// The implementation also handles framerate limiting according to the current
// incoming TV signal. For debugging purposes, the framerate can also be set to
// a specific value
//
// Frame sizing is also handled by the television package. Current information
// about the frame can be acquired with GetFrameInfo(). FrameInfo will also be
// sent to the PixelRenderers as appropriate.
//
// # Simple Television
//
// For backwards compatability and for applications that want simplified
// graphical output, a so-called 'simple television' can be created with the
// NewTelevisionSimple() function.
//
// (I considered implemented the simple television as an entirely separate type
// and to introduce a Television interface at the VCS level. However, this
// seemed too disruptive at first flush and for relatively little gain)
//
// The simple television does not have frame sizing but for the applications
// that are expected to require the simple television, it is not thought these
// features are required.
//
// The VSYNC implementation in the 'simple television is very forgiving and will
// cause no screen roll.
//
// # Concurrency
//
// None of the functions in the Television type are safe to be called from
// goroutines other than the one the type was created in.
//
// # Logging
//
// The television does no logging. This is because the television can be used
// ephemerally and logging would be noisy. Callers of television package
// functions should decide whether it is appropriate to log.
//
// # Struct Embedding
//
// It is possible for instances of television to be embedded in other type
// structure, thereby extending the "features" of the television and allowing
// the extended type to be used wherever the Television interface is required.
// The digest package is a good example of this idea.ckage television
//
// # Compatability with Stella
//
// Note that the television implementation no longer attempts to report the
// same frame/scanline/clock information as Stella. Early versions of the
// implementation did because it facilitated A/B testing but since we're
// now confident that the TIA emulation is correct the need to keep in "sync"
// with Stella is no longer required.
package television
