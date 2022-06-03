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

// Package arm imlplements the ARM7TDMI instruction set as defined in the
// ARM7TDMI Instruction Set Reference:
//
// http://www.ecs.csun.edu/~smirzaei/docs/ece425/arm7tdmi_instruction_set_reference.pdf
//
// For this project we only need to emulte the Thumb architecture. The strategy
// for this was to implement the nineteen opcode formats. As of writing only
// format 17, software interrupts, remain unimplemented. To this end, the
// following reference was preferred:
//
// http://bear.ces.cwru.edu/eecs_382/ARM7-TDMI-manual-pt1.pdf
//
// More detailed explanations of Thumb instruction were found in chapter A7.1
// of the ARM Architecture Reference Manual. In particular the side-effects of
// particular instructions were found in the supplied pseudo-code. Where
// appropriate, the pseudo-code has been included as a comment in the Go
// source.
//
// https://www.cs.miami.edu/home/burt/learning/Csc521.141/Documents/arm_arm.pdf
//
// Reference for the ARM7TDMI-S, as used in the Harmony cartridge formats. This
// contains the cycle information for all ARM instructions.
//
// https://developer.arm.com/documentation/ddi0234/b
//
// Specific information about the NXP ARM7TDMI-S used by the Harmony cartridge.
// This contains good information about the MAM.
//
// https://www.nxp.com/docs/en/user-guide/UM10161.pdf
//
// And the errata, explaining a bug in the MAM that is experienced in the
// some versions of the Harmony cartridge.
//
// https://www.nxp.com/docs/en/errata/ES_LPC2103.pdf
package arm
