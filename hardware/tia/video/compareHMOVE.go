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
	// for the longest time I thought the above note was saying the following:
	//
	// a&0x08 == b&0x08 || a&0x04 == b&0x04 || a&0x02 == b&0x02 || a&0x01 == b&0x01
	//
	// but in practice this isn't the case. the more obvious construct below
	// seems to be the correct interpretation.
	return a&b&0x0f != 0

	// we can see the difference between the two methods in 'Mignight Magic',
	// the separator between the two left-hand gutters,
	// and 'Fatal Run' intro screen
}
