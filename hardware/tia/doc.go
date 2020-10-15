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

// Package TIA implements the custom video/audio chip found in the the VCS. The
// TIA is an extremely tricky device and great effort has been expended in
// creating an accurate emulation. There are undoubtedly subtleties not
// considered by the emulation but none-the-less it seems accurate for a great
// many of the ROMs that have been tested.
//
// For clarity the emulation is split across six packages, which coordinates
// the other five. The principle of the TIA emulation is as follows:
//
// Three times for every tick of the CPU, the TIA Step() function is called.
// The Step() function takes a single argument saying whether TIA memory should
// be checked for side-effects. The timing of this is handled by the VCS type
// in the hardware package but generally speaking there is three video cycles
// (TIA Step()s) for every one CPU cycle.
//
// The best description of how the TIA works is to be found in the document:
//
//	Atari 2600 TIA Hardware Notes, v1.0, by Andrew Towers
//
// The file is often named TIA_HW_Notes.txt and that is the label that is
// used when referring to it, throughout the commentary in this emulation. The
// remainder of this Overview relates the high level concepts described in that
// document with the emulation. Sub-package documentation go into further
// detail, and code commentary into even more.
//
// The two-phase clock generator is implemented in the phaseclock package. Each
// phase clock is ticked forward whenever the part of the TIA affected is
// active. For the Horizontal Clock, the phase clock is ticked forward on every
// call to Step().
//
// Closely related to the phase clock is the polynomial counter or polycounter
// (and found in the package of that name). A polycounter is ticked forward
// whenever its controlling phase clock reaches the rising edge of the second
// phase. In this emulation that means whenever the phase clock's Phi2()
// function returns true.
//
// The most important use of the polycounter is the HSYNC counter. The HSYNC
// counter controls the horizontal scanline of the television screen. At
// key points in the polycounter sequence, signals are sent to the television.
// and are used to synchronise the VCS and the TV (eg. HBLANK)
//
// Updating of TIA registers happens in carefully orchestrated cascade.  I
// defer to the code and commentary for the fuller description of what is
// happening, but it is  sufficient to say here that some side effects must
// take place before others. The various Update*() functions in the tia package
// and sub-packages help achieve this.
//
// At the end of every Step() function (the end of the video cycle) the TV
// signal is compiled and sent to the attached television.
//
// The audio signal is also calculated at this point and sent along at the same
// time. The VCS audio implementation can found in the audio sub-package.
//
// A very important concept in the emulation of the TIA is the concept of
// delays. Delays occur in the TIA as a consequence of the electric circuit
// design and also because of the presence of digital latches. For the
// emulation I have not considered the difference causes in too much detail and
// have implemented delays in the future package. The TIA code is sprinkled
// with references to future Events throughout all packages and sub-packages;
// the timing of when these Events are ticked forward is critically important
// to the accuracy of the emulation. Again, I defer to the code and commentary
// for detailed explanations.
package tia
