package video

// these constants represent the number of cycles required to perform certian
// operations on graphical elements. most operations occur immediately but a
// handful take more than one cycle. for instance, enabling a sprite (the ball
// and missiles) takes one cycle, whereas changing the graphical data of a
// player sprite occurs immediately.
//
// the functions that make use of these values are in the file relevent to the
// graphical element type. these functions in turn call the schedule() function
// of a current instance of future type

// NOTE:
// the document, "Atari 2600 TIA Hardware Notes" by Andrew Towers, talks
// about something called a motion clock which, according to the document,
// is an "inverted (out of phase) CLK signal". rather than emulate the
// motion clock, I think the following delays have the same effect

const (
	delayEnableBall     = 1
	delayEnableMissile  = 1
	delayWritePlayer    = 1
	delayWritePlayfield = 5

	delayResetBall    = 4
	delayResetMissile = 4
	delayResetPlayer  = 4

	delayResetBallHBLANK    = 2
	delayResetMissileHBLANK = 2
	delayResetPlayerHBLANK  = 2
)
