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

package television

// NudgeFPSLimiter stops the FPS limiter for the specified number of frames. A value
// of zero (or less) will stop any existing nudge
func (tv *Television) NudgeFPSLimiter(frames int) {
	if frames < 0 {
		frames = 0
	}
	tv.lmtr.Nudge.Store(int32(frames))
}

// SetFPSLimit whether the emulation should wait for FPS limiter. Returns the
// setting as it was previously.
func (tv *Television) SetFPSLimit(limit bool) bool {
	prev := tv.lmtr.Active
	tv.lmtr.Active = limit

	// notify all pixel renderers that are interested in the FPS limiter
	for i := range tv.renderers {
		if r, ok := tv.renderers[i].(PixelRendererFPSLimiter); ok {
			r.SetFPSLimit(limit)
		}
	}

	return prev
}

// SetFPS requests the number frames per second. This overrides the frame rate of
// the specification. A negative value restores frame rate to the ideal value
// (the frequency of the incoming signal).
func (tv *Television) SetFPS(fps float32) {
	tv.lmtr.SetLimit(fps)
}

// GetIdealFPS returns the ideal number of frames per second. Compare with GetActualFPS() to check for
// accuracy and if the emulation is achieving full speed
//
// IS goroutine safe.
func (tv *Television) GetIdealFPS() float32 {
	return tv.lmtr.IdealFPS.Load().(float32)
}

// GetActualFPS returns the current number of frames per second and the
// detected frequency of the TV signal.
//
// Note that FPS measurement still works even when the frame limiter is disabled.
//
// IS goroutine safe.
func (tv *Television) GetActualFPS() (float32, float32) {
	return tv.lmtr.Measured.Load().(float32), tv.lmtr.RefreshRate.Load().(float32)
}
