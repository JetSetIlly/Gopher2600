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

// Package audio implements the audio generation of the TIA. Originally,
// Gopher2600 used Ron Fries' audio implementation but this has now been
// completely superceded by the work of Chris Brenner.
//
// Chris's work has been used in both the 6502.ts and Stella projects. Source
// from both projects was used as reference. Both projects are exactly
// equivalent.
//
//	  6502.ts (published under the MIT licence)
//			https://github.com/6502ts/6502.ts/blob/6f923e5fe693b82a2448ffac1f85aea9693cacff/src/machine/stella/tia/PCMAudio.ts
//			https://github.com/6502ts/6502.ts/blob/6f923e5fe693b82a2448ffac1f85aea9693cacff/src/machine/stella/tia/PCMChannel.ts
//
//	 Stella (published under the GNU GPL v2.0 licence)
//			https://github.com/stella-emu/stella/blob/e6af23d6c12893dd17711002971087f28f87c31f/src/emucore/tia/Audio.cxx
//			https://github.com/stella-emu/stella/blob/e6af23d6c12893dd17711002971087f28f87c31f/src/emucore/tia/AudioChannel.cxx
//
// Additional work on volume sampling a result of this thread:
//
// https://forums.atariage.com/topic/370460-8-bit-digital-audio-from-2600/
//
// For reference, Ron Fries' audio method is represented here:
//
// https://raw.githubusercontent.com/alekmaul/stella/master/emucore/TIASound.c
package audio
