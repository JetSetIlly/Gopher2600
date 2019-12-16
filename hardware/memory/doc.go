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
//	                           ---- TIA ---- chip bus ---- TIA
//	                          |
//	                          |---- RIOT ---- chip bus ---- RIOT
//	    CPU ---- cpu bus ---- *
//	                          |---- PIA RAM
//	                          |
//	                           -<-- Cartridge
//
//
// The asterisk indicates that addresses used by the CPU are mapped to the
// primary address. The memorymap package contains more detail on this.
//
// The arrow pointing away from the Cartridge area indicates that the CPU can
// only read from the cartridge, it cannot write to it. Unless that is, the
// cartridge has internal RAM. See the cartridge package documentation for the
// discussion on this.
//
// Note that the RIOT registers and PIA RAM are all part of the same hardware
// package, the PIA 6532. However, for our purposes the two memory areas are
// considered to be entirely separate.
package memory
