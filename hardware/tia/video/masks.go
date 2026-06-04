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

package video

// Not all bits in TIA graphic registers are used. The following masks can be
// be used to keep only the relevant bits from the value that has been written
// to the register.
const (
	CTRLPFPriorityMask  uint8 = 0x04
	CTRLPFScoremodeMask uint8 = 0x02
	CTRLPFReflectedMask uint8 = 0x01
	REFPxMask           uint8 = 0x08
	VDELPxMask          uint8 = 0x01
	RESMPxMask          uint8 = 0x02
	ENAxxMask           uint8 = 0x02
	HMxxMask            uint8 = 0xf0
	NUSIZxCopiesMask    uint8 = 0x07
	NUSIZxSizeMask      uint8 = 0x03
)
