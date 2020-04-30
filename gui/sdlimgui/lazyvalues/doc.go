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

// Package lazyvalues is the method used by sdlimgui (and possibly other GUI
// implementations) when access emulator data from the GUI thread. Accessing
// emulator values will cause race errors in almost every circumstance so it is
// important that this lazyvalue mechanism be used whenevert emulator
// information is required.
//
// Note that this system is used in addition to the other systems which hand
// off information to the GUI. The PixelRenderer and AudioMixer interfaces from
// the television packages should be used in the normal way. The terminal
// interface is also available and should be used to send and recieve responses
// from the underlying debugger. The lazyvalues system is for information that
// is not available through either of those systems or which would be too slow
// to retrieve through the terminal.
//
// Reading values from the emulator can be done through the Lazy types and/or
// through one of the sub-types. For example, retrieving the foreground color
// of the playfield:
//
//  fgCol := lazyval.Playfield.ForegroundColor
//
// Note that some values require additional context and are wrapped as
// functions. For example, reading RAM is done through the ReadRAM() function.
//
// When writing values directly to the emulator, the GUI thread must do so
// through the debugger's RawEvent queue. For example:
//
//	lazyval.Dbg.PushRawEvent(func() { lazyval.VCS.TIA.Video.Playfield.ForegroundColor = fgCol })
//
// Note that the Debugger and VCS instances are exposed by the Lazy type in
// this package but these *must not* be used except through PushRawEvent.
//
// Because of the nature of the lazyvalues system, variable scope needs to be
// considered. As a rule, if a value retrieved from the lazy system is to be
// altered, then make a copy of that value before doing so. If it is only for
// presentation purposes, then a copy probably does not need to be made.
//
// By the same token, you should be careful about variable reuse. Do not be
// tempted by the following pattern
//
//  col := lazyval.Playfield.ForegroundColor
//
//  <update foreground color with PushRawEvent()>
//
//  col = lazyval.Playfield.BackgroundColor
//
//  <update background color with PushRawEvent()>
//
// Because PushRawEvent will update the values "lazily", by the time the first
// PushRawEvent() has ran the color variable will have been updated with the
// background color value.
package lazyvalues
