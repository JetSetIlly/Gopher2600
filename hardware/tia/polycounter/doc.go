// Package polycounter implements the polynomial counters found in the Atari
// 2600. Described by Andrew Towers in the "Atari 2600 TIA Hardware Notes"
// (TIA_HW_Notes.txt), polynomial counters are a predictably performative way
// of counting in simple electronics - performance of ripple counters can change
// due to carrying etc.
//
// In our emulation we are of course using ripple counters internally. But for
// the purposes of debugging the TIA loop (HSYNC counter) we'd still like to
// know what the equivalent polycounter value is. We use a 6-bit polycounter
// for this.
//
//		hsync := polycounter.New(6)
//
// We advance the counter with the Tick() function. The binary representation
// of the polycounter value can be retreived with the ToBinary() function.
//
// In the 2600, polycounter logic is also used to generate the bit sequences
// required for TIA audio emulation. A real TIA variously uses 4-bit, 5-bit and
// 9-bit polycounters to generate the sound waves available to the 2600
// programmer. As of yet, this package doesn't support this functionality
// correctly. The bit sequences required are hard-coded into the tia/audio
// package as discovered by Ron Fries.
package polycounter
