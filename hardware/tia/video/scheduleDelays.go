package video

// these constants represent the number of cycles required to perform certian
// operations on sprites.
//
// in all cases, these values have been determined by observation and by
// technical commentary, such as Andrew Tower's "Atari 2600 TIA Hardware
// Notes" (TIA_HW_Notes.txt).
//
// see Future type, the schedule() function in paricular, to see how these
// values are used.

const (
	delayEnableBall     = 2
	delayEnableMissile  = 2
	delayWritePlayer    = 2
	delayWritePlayfield = 3

	// TIA_HW_Notes.txt: "there are 5 CLK worth of clocking/latching to take
	// into account"
	delayResetBall    = 5
	delayResetMissile = 5
	delayResetPlayer  = 5
)
