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

// During development an internal alternative to the CPUBus was considered (see
// bus package). The idea was to force use of mapped address when required.
// This would require new type, MappedAddr, which MapAddress() would return a
// value of. The MappedCPUBus in turn would expect address values of that type.
// However, after some experimentation the idea was deemed to be too clumsy and
// didn't help in clarification. If the memory implementation was required to
// be more general then it would be a good idea but as it is, it is not
// necessary.
package memorymap
