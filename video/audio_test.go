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

package video

import (
	"testing"

	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/test"
)

const testSamplesPerFrame = 524

// frames of the specification's length give every sample unchanged
func TestFitSoundUnchanged(t *testing.T) {
	sound := make([]int16, 100*testSamplesPerFrame)
	for i := range sound {
		sound[i] = int16(i * 7919)
	}
	bounds := make([]int, 51)
	for i := range bounds {
		bounds[i] = 1000 + i*testSamplesPerFrame
	}
	out, minLength, maxLength := fitSound(sound, bounds, testSamplesPerFrame)
	test.ExpectEquality(t, len(out), 50*testSamplesPerFrame)
	test.ExpectEquality(t, minLength, float64(testSamplesPerFrame))
	test.ExpectEquality(t, maxLength, float64(testSamplesPerFrame))
	for i := range out {
		if !test.ExpectEquality(t, out[i], sound[1000+i], i) {
			break
		}
	}
}

// testTV calls the audio type's functions as the television does: two samples on every scanline,
// sent in batches of about a frame that do not start or end with a frame. the first sample of every
// frame is loud and the others are silent
type testTV struct {
	au       *audio
	pending  []signal.AudioSignalAttributes
	batch    int
	frameNum int
}

func (tv *testTV) frame(scanlines int) {
	for sl := range scanlines {
		for s := range 2 {
			var vol uint8
			if sl == 0 && s == 0 {
				vol = 15
			}
			tv.pending = append(tv.pending, signal.AudioSignalAttributes{AudioChannel0: vol})
			if len(tv.pending) >= tv.batch {
				tv.au.SetAudio(tv.pending)
				tv.pending = tv.pending[:0]
			}
		}
		if sl < scanlines-1 {
			tv.au.NewScanline(frameinfo.Current{})
		} else {
			tv.au.NewFrame(frameinfo.Current{FrameNum: tv.frameNum})
			tv.au.addPicture(tv.frameNum)
			tv.frameNum++
			tv.batch = scanlines * 2
		}
	}
}

// fitFrames records frames of the given lengths and checks that the start of each frame's sound
// is within tolerance samples of the start of its picture
func fitFrames(t *testing.T, scanlines func(int) int, frames int, tolerance int) {
	t.Helper()

	// the recording starts part way through a batch of audio
	tv := testTV{
		au:      &audio{samplesPerFrame: testSamplesPerFrame},
		pending: make([]signal.AudioSignalAttributes, 100),
		batch:   testSamplesPerFrame,
	}
	for i := range frames {
		tv.frame(scanlines(i))
	}
	out := tv.au.fit()
	test.ExpectEquality(t, len(out), frames*testSamplesPerFrame)
	test.ExpectEquality(t, tv.au.fitted.noSound, 0)

	// the start of the first frame is estimated, so it is not checked
	for i := 1; i < frames; i++ {
		centre := i * testSamplesPerFrame
		peak := centre - tolerance - 1
		for j := centre - tolerance - 1; j <= centre+tolerance+1; j++ {
			if out[j] > out[peak] {
				peak = j
			}
		}
		if peak < centre-tolerance || peak > centre+tolerance {
			t.Errorf("frame %d: sound starts at sample %d, picture at %d", i, peak, centre)
			break
		}
	}
}

func TestFitFrames(t *testing.T) {
	// the position of a frame end is right to within a sample
	fitFrames(t, func(int) int { return 262 }, 300, 0)
	fitFrames(t, func(int) int { return 263 }, 300, 1)
	fitFrames(t, func(int) int { return 260 }, 300, 1)

	// for a wobbling number of scanlines the stretch follows the average, so the start of a
	// frame's sound moves by as much as the wobble
	fitFrames(t, func(i int) int { return 262 + 2*(i%2) }, 300, 3)
	extra := []int{0, 3, 7, 1, 10, 4, 0, 8, 2, 6, 9, 1, 5, 3, 10, 7}
	fitFrames(t, func(i int) int { return 258 + extra[i%16] }, 300, 24)
}

// a picture whose frame has no recorded sound is given silence, and the pictures after it still
// have their own sound
func TestFitNoSound(t *testing.T) {
	tv := testTV{
		au:    &audio{samplesPerFrame: testSamplesPerFrame},
		batch: testSamplesPerFrame,
	}
	for range 10 {
		tv.frame(262)
	}
	tv.au.pictures = []int{1, 2, 99, 3, 4}
	out := tv.au.fit()
	test.ExpectEquality(t, len(out), 5*testSamplesPerFrame)
	test.ExpectEquality(t, tv.au.fitted.noSound, 1)
	for i := 2 * testSamplesPerFrame; i < 3*testSamplesPerFrame; i++ {
		if !test.ExpectEquality(t, out[i], 0, i) {
			break
		}
	}
	test.ExpectInequality(t, out[3*testSamplesPerFrame], 0)
	test.ExpectInequality(t, out[4*testSamplesPerFrame], 0)
}
