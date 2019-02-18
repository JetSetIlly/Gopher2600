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
	delayWritePlayfield          = 3
	delayWritePlayer             = 2
	delayEnableBall              = 2
	delayEnableMissile           = 2
	delayResetMissileToPlayerPos = 2
	delayVDELBL                  = 1
	delayVDELP                   = 1
	delayNUSIZ                   = 3

	// TIA_HW_Notes.txt: "there are 5 CLK worth of clocking/latching to take
	// into account". not entirely sure this passage is relevant to my solution
	// or if it's just a coincidence.
	delayResetMissile = 5
	delayResetPlayer  = 5
	delayResetBall    = 5

	// see comment in sprite.tickSpritesForHMOVE()
	delayForceReset = 3
)
