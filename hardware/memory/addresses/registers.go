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

// ChipRegister specifies the offset of a chip register in the chip memory
// areas. It is used in contexts where a register is required, as opposed to an
// address.
type ChipRegister int

// TIA registers
//
// These value are used by the emulator to specify known addresses. For
// example, when writing collision information we know we need the CXM0P
// register. these named values make the code more readable
//
// Values are enumerated from 0; value is added to the origin address of the
// TIA in ChipBus.ChipWrite implementation.
const (
	CXM0P ChipRegister = iota
	CXM1P
	CXP0FB
	CXP1FB
	CXM0FB
	CXM1FB
	CXBLPF
	CXPPMM
	INPT0
	INPT1
	INPT2
	INPT3
	INPT4
	INPT5
)

// RIOT registers
//
// These value are used by the emulator to specify known addresses. For
// example, the timer updates itself every cycle and stores time remaining
// value in the INTIM register.
//
// Values are enumerated from 0; value is added to the origin address of the
// TIA in ChipBus.ChipWrite implementation.
const (
	SWCHA ChipRegister = iota
	SWACNT
	SWCHB
	SWBCNT
	INTIM
	TIMINT
)
