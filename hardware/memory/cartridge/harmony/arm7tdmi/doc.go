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
// ARM Architecture Reference Manual:
//
// https://www.cs.miami.edu/home/burt/learning/Csc521.141/Documents/arm_arm.pdf
//
// Harmony DPC+ ARM:
//
// https://atariage.com/forums/blogs/entry/11712-dpc-arm-development/?tab=comments#comment-27116
// https://atariage.com/forums/topic/163834-harmony-dpc-arm-programming/
//
package arm7tdmi
