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

package elf

// the origin address of argument memory. memtop is defeined beloew
const (
	argOrigin uint32 = 0xffffff00
)

// the addresses in memory of each argument sent to the main function the elf
// program. the final entry in the list is the memtop of argument memory
const (
	argAddrSystemType = argOrigin + (iota * 4)
	argAddrClockHz
	argAddrFlags
	argAddrElapsed
	argAddrThreshold
	argMemtop
)

// supported values for argAddrSystemType
const (
	argSystemType_NTSC = iota
	argSystemType_PAL
	argSystemType_PAL60
)

// supported values for argAddrFlags
const (
	argFlags_NoExit = iota

	// indicates that the ELF is loaded by a multicart that supports reentry
	argFlags_ExitToMultiCart
)
