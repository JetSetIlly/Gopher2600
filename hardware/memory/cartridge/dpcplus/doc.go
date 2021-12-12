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

// Package dpcplus implements the DPC+ cartridge mapper. It was developed by
// adapting the existing DPC mapper, which is well documented. Differences to
// this extended mapper were learned by studying the following URLs.
//
// https://atariage.com/forums/blogs/entry/11712-dpc-arm-development/?tab=comments#comment-27116
//
// https://atariage.com/forums/topic/163834-harmony-dpc-arm-programming/
//
// The only DPC+ ROMs that I am aware of that don't use the Harmony ARM
// coprocessor is, Chaotic Grill and the ROM titled "DPC+demo.bin" by Darrell
// Sprice.
package dpcplus
