package video

// CompareHMOVE tests to variables of type uint8 and checks to see if any of
// the bits in the lower nibble differ. returns false if no bits are the same,
// true otherwise
//
// returns true if any corresponding bits in the lower nibble are the same.
// from TIA_HW_Notes.txt:
//
// "When the comparator for a given object detects that none of the 4 bits
// match the bits in the counter state, it clears this latch"
//
func compareHMOVE(a uint8, b uint8) bool {
	return a&0x08 == b&0x08 || a&0x04 == b&0x04 || a&0x02 == b&0x02 || a&0x01 == b&0x01

	// at first sight TIA_HW_Notes.txt seems to be saying "a&b!=0" but after
	// some thought, I don't believe it is.
	//
	//return a&b != 0
}
