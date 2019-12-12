// Package cartridge fully implements loading of mapping of cartridge memory.
//
// There are many different types of cartridge most of which are supported by
// the package. Some cartridge types contain additional RAM but the main
// difference is how they map additional ROM to the relatively small address
// space available for cartridges in the VCS. This is called bank-switching.
// All of these differences are handled transparently by the package.
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
package cartridge
