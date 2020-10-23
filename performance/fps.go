// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package performance

import "github.com/jetsetilly/gopher2600/hardware/television"

// CalcFPS takes the the number of frames and duration (in seconds) and returns
// the frames-per-second and the accuracy of that value as a percentage.
func CalcFPS(tv *television.Television, numFrames int, duration float64) (fps float64, accuracy float64) {
	fps = float64(numFrames) / duration
	spec := tv.GetSpec()
	accuracy = 100 * float64(numFrames) / (duration * float64(spec.FramesPerSecond))
	return fps, accuracy
}
