// Package memorymap facilitates the translation of addresses to primary
// address equivalents.
//
// Because of the limited number of address lines used by the 6507 in the VCS
// the number of addressable locations is a lot less than the 16bit suggested
// by the addressing model of the CPU. The MapAddress() functions should be
// used to produce a "mapped address" whenever an address is being used from
// the viewport of the CPU. (Writing to memory from the viewpoint of TIA & RIOT
// is completely different)
//
//	ma, _ := memorymap.MapAddress(address, true)
//
// The second argument indicates if the address is being read or being written
// to. Some addresses require an additional transformation if they are being
// read. Again, the details are handled by the function.
package memorymap
