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

// Package arm7tdmi imlplements the ARM7TDMI instruction set as defined in the
// ARM7TDMI Instruction Set Reference:
//
// http://www.ecs.csun.edu/~smirzaei/docs/ece425/arm7tdmi_instruction_set_reference.pdf
//
// For this project we only need to emulte the Thumb architecture. The strategy
// for this was to implement the ninetween opcode formats in turn, until there
// was nothing left. As of writing only format 17, software interrupts, remain
// unimplemented. To this end, the following reference was preferred:
//
// https://usermanual.wiki/Pdf/ARM7TDMImanualpt3.1481331792/view
//
// More detailed explanations of Thumb instruction were found in chapter A7.1
// of the ARM Architecture Reference Manual. In particular the side-effects of
// particular instructions were found in the supplied pseudo-code. Where
// appropriate, the pseudo-code has been included as a comment in the Go
// source.
//
// https://www.cs.miami.edu/home/burt/learning/Csc521.141/Documents/arm_arm.pdf
//
package arm7tdmi
