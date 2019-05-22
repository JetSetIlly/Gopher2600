package delay

// these constants represent the number of cycles required to perform certain
// operations on sprites.
//
// in all cases, these values have been determined by observation and from
// technical commentary, in particular, Andrew Tower's "Atari 2600 TIA Hardware
// Notes" (TIA_HW_Notes.txt).
const (
	// value revealed by the scoring box of homebrew donkey kong
	TriggerVBLANK = 2

	// playfield write delay of two cycles is optimal - the effect of three
	// cycles is hard to see but can be clearly shown with homebrew Thrust
	WritePlayfield = 2

	// the almost certain value of a two cycle delay for writing playfield data
	// points us to a similar delay for other events
	WritePlayer             = 2
	EnableBall              = 2
	EnableMissile           = 2
	ResetMissileToPlayerPos = 2

	SetVDELBL = 1
	SetVDELP  = 1
	SetNUSIZ  = 3

	// TIA_HW_Notes.txt: "there are 5 CLK worth of clocking/latching to take
	// into account". not entirely sure this passage is relevant to my solution
	// or if it's just a coincidence.
	ResetMissile = 5
	ResetPlayer  = 5
	ResetBall    = 5

	// see comment in sprite.resolveHorizMovement()
	ForceReset = 3
)
