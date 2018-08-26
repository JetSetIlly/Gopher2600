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

const (
	delayEnableBall     = 2
	delayEnableMissile  = 2
	delayWritePlayer    = 2
	delayWritePlayfield = 5

	delayResetBall    = 5
	delayResetMissile = 5
	delayResetPlayer  = 5
)
