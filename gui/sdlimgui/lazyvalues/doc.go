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

// Package lazyvalues is the method used by sdlimgui to read emulator data
// from the GUI thread. Reading emulator values (which are handled by the
// emulator thread) will cause race errors in almost every circumstance so it
// is important that this lazyvalue mechanism be used whenever emulator
// information is required.
//
// Note that this system is used in addition to the other systems which hand
// off information to the GUI. The PixelRenderer and AudioMixer interfaces from
// the television packages should be used in the normal way.
//
// For writing data back to the emulation thread the terminal interface can be
// used for many things. Alternatively the debugger.PushRawEvent() function can
// be used. There is currently no way of pushing events onto the emulator
// unless the debugging loop is in use.
//
//
// Example
// -------
//
// Retrieving the foreground color of the playfield:
//
//  col := lazyval.Playfield.ForegroundColor
//
//
// Writing the playfield values is done thought debugger's "raw event" system:
//
//	lazyval.Dbg.PushRawEvent(func() {
//		lazyval.VCS.TIA.Video.Playfield.ForegroundColor = col
//	})
//
//
// Implementation
// --------------
//
// The main goal of the lazyvalues system is to prevent the GUI loop from
// locking up while waiting for a response from the emulator thread. Given that
// we must use a thred-sage a communication channel between the GUI and
// emulator threads to avoid race conditions this is important - a unresponsive
// GUI can needlessly damage the user experience.
//
// This section outlines the principles of the internals of the lazyvalues
// package. Users of the package need not understand these points.
//
// The principle of the lazyvalues system is to use whatever values are available
// immediately and to update those values "lazily". In a GUI context this means
// that the values seen on screen may be several frames behind the emulation
// but at normal GUI refresh rates this isn't noticeable. Cartainly, when the
// emulation is paused, the values seen in the GUI will be accurate.
//
// Lazy values are updated with the Refresh() function. In turn, this function
// will call the push() and update() functions of each component in the
// lazyvalues package.
//
// The pseudocode below shows how the Refresh() updates the values in every
// type in the lazyvalues system, at the same time as requesting new values.
//
//	func Refresh() {                        .------------------.
//		debugger.PushRawEvent()   ----->	| CPU.push()       |
//											| RAM.push()       |
//      CPU.update()						| Playfield.push() |
//		RAM.update()						|   .              |
//			.								|   .              |
//			.								|   .              |
//			.								| Log.push()       |
//		Log.update()						 ------------------
//	}
//
// The update() and push() functions (not visible from outside the lazyvalues
// package) of each type handle the retreiving and updating of emulation
// values. In most instances, this is achieved with the atomic.Value type, from
// the atomic package in the Go standard library.
//
// In the instance of the LazyController type, we cannot use the atomic.Value.
// This is because of the limitation on atomic.Values only being able to store
// consistently typed values. In the case of the LazyController type we need to
// store the ports.Peripheral interface, which by definition may have differing
// underlying types.
//
// For this reason, the LazyController type uses channels to communicate
// between the push() function (ie. the emulation thread) and the update()
// function, rather than atomic values. We could of course, use channels for
// all types and do away with atomic values but it is felt that in most cases
// the atomic solution is clearer.
//
// As a final point about atomic values, note that arrays of atomic values
// require that the array itself be an atomic value, as well as the elements of
// the array. For example, the RAM package has code equivalent to this; an
// array of atomic.Value stored as an atomic value:
//
//	 var ram atomic.Value
//   ram.Store(make([]atomic.Value, size)
//
// The exception to all the rules is the LazyBreakpoints type. Like LazyRAM it
// employs an array of atomic.Values storied as an atomic Value but unlike
// everythin else it is not refreshed with update() and push(). Instead, the
// unique function HasBreak() is used, which is called by the Disassembly
// window for every cartridge entry that is visible.
//
// The reason for this function is so that we can pass an instance of
// disassembly.Entry and probe the debugger's breakpoints with that. There may
// be other ways of achieving the same effect, but whatever way we do it the
// additional context provided by the disassembly.Entry is required.
//
package lazyvalues
