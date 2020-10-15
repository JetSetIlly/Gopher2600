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

package addresses

// DataMasks are applied to data read by the CPU from lowest 16 addresses. This
// requirement is a consequence of how the address/data bus works in the VCS.
//
// For example, if the CPU wants to read the contents of the CXM1P register, it
// can use the address 0x0d to do so.
//
//		LDA 0x01
//
// If there are no collisions (between missile 1 and either player, in this
// case) than the value of the most significant bits are zero. The lower six
// bits are not part of the CXM1P register and are left undefined by the TIA
// when the data is put on the bus. The lower bits of the LDA operation are in
// fact "left over" from the address. In our example, the lowest six bits are
//
//		0bxx000001
//
// meaning the the returned data is in fact 0x01 and not 0x00, as you might
// expect.  Things get interesting when we use mirrored addresses. If instead
// of 0x01 we used the mirror address 0x11, the lowest six bits are:
//
//		0bxx01001
//
// meaning that the returned value is 0x11 and not (again, as you might expect)
// 0x00 or even 0x01.
//
// So what happens if there is sprite collision information in the register?
// Meaning that the top bits are not necessarily zero. Let's say there is a
// collusion between missile 1 and player 0, the data before masking will be
//
//		0b01000000
//
// If we used address 0x11 to load this value, we would in fact, get this
// pattern (0x51 in hex):
//
//		0b01010001
//
// Now, if all ROMs read and interpreted chip registers only as they're
// supposed to (defails in the 2600 programmer's guide) then none of this would
// matter but some ROMs do make use of the extra bits, and so we must account
// for it in emulation.
//
// It's worth noting that the above is implicitly talking about zero-page
// addressing; but masking also occurs with regular two-byte addressing. The
// key to understanding is that the masking is applied to the most recent byte
// of the address to be put on the address bus*. In all cases, this is the
// most-significant byte. So, if the requested address is 0x171, the bit
// pattern for the address is:
//
//		0x0000000101110001
//
// the most significant byte in this pattern is 0x00000001 and so the data
// retreived is AND-ed with that. The mapped address for 0x171 incidentally, is
// 0x01, which is the CXM1P register also used in the examples above.
//
var DataMasks = []uint8{
	0b11000000, // CXM0P
	0b11000000, // CXM1P
	0b11000000, // CXP0FB
	0b11000000, // CXP1FB
	0b11000000, // CXM0FB
	0b11000000, // CXM1FB

	// event though legitimate usage of CXBLPF suggests only the most
	// significant bit is used, for the purposes of masking it acts just like
	// the other collision registers
	0b11000000, // CXBLPF

	0b11000000, // CXPPMM
	0b10000000, // INPT0
	0b10000000, // INPT1
	0b10000000, // INPT2
	0b10000000, // INPT3
	0b10000000, // INPT4
	0b10000000, // INPT5

	// the contents of the last two locations are "undefined" according to the
	// Stella Programmer's Guide but are readable anyway. we can see through
	// experiementation that the mask is as follows (details of what we
	// experimented with has been forgotten)
	0b11000000,
	0b11000000,
}
