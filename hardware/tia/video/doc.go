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

// Package video implements pixel generation for the emulated TIA. Pixel
// generation is conceptually divided into six areas, implemented as types.
// These are:
//
//	Playfield
//	Player 0  and  Player 1
//	Missile 0  and  Missile 1
//	Ball
//
// Collectively we can refer to these as the playfield and sprites (even
// though the VCS sprites are nothing like what we now think of sprites, it is
// a useful appellation none-the-less).
//
// The video subsystem is ticked along with the TIA every video cycle. The
// playfield is closely related to the TIA's HSYNC and is not ticked
// separately. The sprites are ticked depending on the state of the TIA's
// HBLANK signal; it also depends on whether HMOVE has been recently latched in
// the TIA and whether the sprite has completed any horizontal movement. For
// this reason the video sub-system and the sprites are initialised with
// references to the HBLANK signal and the HMOVE latch.
//
// TIA memory registers of direct interest to the video subsystem are read and
// divided into six different Update*() functions. The timing of when video
// information is updated is important and dividing the update functions in
// this manner helps. The TIA package handles these timings and whether they
// need to be called at all.
//
// The three sprite categories, player, missile and ball, all have common
// features but have been implemented to be completely separate from one
// another. The exception is the enclockifier type used by both missiles and
// the ball. All implementatio decisions of this type have been made for
// reasons of clarity.
//
// All sprites keep track of their own phase clocks and position counters.
// Delayed side effects only occur when the sprite itself is ticked and so each
// sprite also has an instance of Ticker from the future package.
//
// A significant difference to the description in Andrew Towers' document,
// "Atari 2600 TIA Hardware Notes", is how HMOVE counters are handled. In
// Towers'  description of the hardware, the HMOVE latch, the counters and the
// signal line to the sprite are all intertwined. In the emulation this is
// almost turned inside out with the sprite maintaining its own counter and
// ticking (include HMOVE stuffing ticks) only when required.
//
// Somewhere during the cycle the video sub-system will decide on what the
// pixel output should be. In this context we strictly mean VCS pixels.  That
// is, we're deciding what the colour of all TV pixels for the duration of the
// video cycle should be.
//
// The timing of this decision is critical: it must happen before some register
// updates but after others. Note that the pixel color decision is distinct
// from sending the color signal to the TV (which is handled by the TIA)
// package). Sending of the color signal always happens at the very end of the
// video cycle.
//
// To effectively make the pixel color decision, the video sub-system at the
// same time process the pixel priority. For convenience, pixel collisions are
// also set at this time.
package video
