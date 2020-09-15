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

// Package savekey implements the SaveKey external memory card. It contains
// 32KB of non-volatile memory. Suitable for saving high scores, game settings,
// etc.
//
// The SaveKey type implements the ports.Peripheral interface and can be
// inserted into a VCS port like any other peripheral.
//
// SaveKey information taken from "AtariVox Programmer's Guide" (16/11/04) by Alex Herbert
//
// i2c information taken from https://learn.sparkfun.com/tutorials/i2c/all (13/09/2020)
//
// allocation list at https://atariage.com/atarivox/atarivox_mem_list.html (13/09/2020)
package savekey
