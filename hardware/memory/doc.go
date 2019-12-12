// Package memory implements the Atari VCS memory model. The addresses and
// memory sub-packages help with this.
//
// It is important to understand that memory is viewed differently by different
// parts of the VCS system. To help with this, the emulation uses what has been
// called memory busses. These busses have nothing to do with the real
// hardware; they are purely conceptual and are implemented through Go
// interfaces.
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
//	                           ---- TIA ---- chip bus ---- TIA
//	                          |
//	                          |---- RIOT ---- chip bus ---- RIOT
//	    CPU ---- cpu bus ---- *
//	                          |---- PIA
//	                          |
//	                           -<-- Cartridge
//
//
// The asterisk indicates that addresses used by the CPU are first mapped to
// the primary address. The memorymap package contains more detail on this.
//
// The arrow pointing away from the Cartridge area indicates that the CPU can
// only read from the cartridge, it cannot write to it. Unless that is, the
// cartridge has internal RAM. The differences in cartridge abilities in this
// regard is handled by the cartMapper interface.
//
// The cartMapper interface allows the transparent implementation of the
// different cartridge formats that have been used by the VCS. We've already
// mentioned cartridge RAM but the major difference betwen cartridge types is
// how they handle so-called bank-switching.
//
// The differences between the cartridge types is too much to go into here but
// a good reference for this can be found here:
//
// http://blog.kevtris.org/blogfiles/Atari%202600%20Mappers.txt
//
// Currently supported cartridge types are:
//
//	- Atari 2k / 4k / 8k / 16k and 32k
//
//	- the above with additional Superchip (additional RAM in other words)
//
//	- Parker Bros.
//
//	- MNetwork
//
//	- Tigervision
//
//	- CBS
//
// Other cartridge types can easily be added using the cartMapper system.
package memory
