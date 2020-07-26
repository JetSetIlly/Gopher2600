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

// Package memory implements the Atari VCS memory model. The addresses and
// memory sub-packages help with this.
//
// It is important to understand that memory is viewed differently by different
// parts of the VCS system. To help with this, the emulation uses what we call
// memory buses. These buses have nothing to do with the real hardware; they
// are purely conceptual and are implemented through Go interfaces.
//
// The following ASCII diagram tries to show how the different components of
// the VCS are connected to the memory. What may not be clear from this diagram
// is that the peripheral bus only ever writes to memory. The other buses are
// bidirectional.
//
//
//	                        PERIPHERALS
//
//	                             |
//	                             |
//	                             \/
//
//	                         periph bus
//
//	                             |
//	                             |
//	                             \/
//
//	    CPU ---- cpu bus ---- MEMORY ---- chip bus ---- TIA
//	                                                \
//	                             |                   \
//	                             |                    \---- RIOT
//
//	                        debugger bus
//
//	                             |
//	                             |
//
//	                          DEBUGGER
//
//
// The memory itself is divided into areas, defined in the memorymap package.
// Removing the periph bus and debugger bus from the picture, the above diagram
// with memory areas added is as follows:
//
//
//	                           ===* TIA ---- chip bus ---- TIA
//	                          |
//	                          |===* RIOT ---- chip bus ---- RIOT
//	    CPU ---- cpu bus -----
//	                          |===* PIA RAM
//	                          |
//	                           ==== Cartridge
//
//
// The asterisk indicates that addresses used by the CPU are mapped to the
// primary mirror address. The memorymap package contains more detail on this.
//
// Cartridge memory is accessed by whatever mirror address the CPU wants. This
// is because some cartridge formats might be sensitive to which mirror is
// being used (eg. Supercharger). Cartridges also implement a Listen()
// function. This is a special function outside of the bussing system. See
// cartridge package for details.
//
// Note that the RIOT registers and PIA RAM are all part of the same hardware
// package, the PIA 6532. However, for our purposes the two memory areas are
// considered to be entirely separate.
package memory
