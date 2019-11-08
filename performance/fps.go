package performance

import "gopher2600/television"

// CalcFPS takes the the number of frames and duration and returns the
// frames-per-second and the accuracy of that value as a percentage.
func CalcFPS(ftv television.Television, numFrames int, duration float64) (fps float64, accuracy float64) {
	fps = float64(numFrames) / duration
	accuracy = 100 * float64(numFrames) / (duration * float64(ftv.GetSpec().FramesPerSecond))
	return fps, accuracy
}
