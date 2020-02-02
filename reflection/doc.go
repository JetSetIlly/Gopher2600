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

// Package reflection monitors the emulated hardware for conditions that would
// otherwise not be visible. In particular it signals the reflection renderer
// when certain memory addresses have been written to. For example, the HMOVE
// register.
//
// In addition it monitors the state of WSYNC and signals the reflection
// renderer when the CPU is idle. This makes for quite a nice visual indication
// of "lost cycles" or potential extra cycles that could be regained with a bit
// of reorgnisation.
//
// There are lots of other things we could potentially do with the reflection
// idea but as it is, it is a little underdeveloped. In particular, it's rather
// slow but I'm not too worried about that because this is for debugging not
// actually playing games and such.
//
// I think the next thing this needs is a way of making the various monitors
// switchable at runtime. As it is, what's compiled is what we get. If we
// monitored every possible thing, a visual display would get cluttered very
// quickly. It would be nice to be able to define groups (say, a player sprites
// group, a HMOVE group, etc.) and to turn them on and off according to our
// needs.
package reflection
