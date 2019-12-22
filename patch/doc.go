// Package patch is used to patch the contents of a cartridge. It works on
// cartridge memory once a cartridge file has been attached. The package does
// not implement the patching directly, rather the different cartridge mappers
// (see cartridge package) deal with that individually.
//
// This package simply loads the patch instructions, interprets them and calls
// the cartridge.Patch() function. Currently only one patch format is
// supported. This is an ad-hoc format taken from the "In case you can't wait"
// section of the following web page:
//
//	"Fixing E.T. The Extra-Terrestrial for the Atari 2600"
//
//	http://www.neocomputer.org/projects/et/
//
// The following extract illustrates the format:
//
//		-------------------------------------------
//		- E.T. is Not Green
//		-------------------------------------------
//		17FA: FE FC F8 F8 F8
//		1DE8: 04
//
// Rules:
//
//	1. Lines beginning with a hyphen or white space are ignored
//	2. Addresses and values are expressed in hex (case-insensitive)
//	3. Values and addresses are separated by a colon
//	4. Multiple values on a line are poked into consecutive addresses, starting
//	    from the address value
//
// Note that addresses are expressed with origin zero and have no relationship
// to how memory is mapped inside the VCS. Imagine that the patches are being
// applied to the cartridge file image. The cartridge mapper handles the VCS
// memory side of things.
package patch
