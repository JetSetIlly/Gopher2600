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
	"math"
	"slices"
	"sync"

	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	tiaaudio "github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

// the sound for each video frame is stretched or squeezed by the average length of the frames
// around it, so that the pitch stays steady when the number of scanlines in each frame wobbles.
// smoothFrames is the number of frames on either side that are included in the average
const smoothFrames = 15

// the end of a frame in the recorded sound
type frameEnd struct {
	// frame number from the television
	frameNum int

	// the number of scanlines since the recording started, including the frame's last scanline
	scanlines int

	// the offset (see audio type) at the end of the frame
	offset      int
	offsetValid bool
}

// summary of how the sound was fitted to the pictures
type fitSummary struct {
	pictures int
	noSound  int

	// the shortest and longest frames after averaging, in samples
	minLength float64
	maxLength float64
}

type audio struct {
	wavs *wavwriter.WavWriter

	// the number of samples in each video frame
	samplesPerFrame int

	// the number of scanlines since the recording started. only used by the emulation goroutine
	scanlines int

	// the fields below are written by the emulation goroutine, except pictures which is written by
	// the GUI goroutine
	crit sync.Mutex

	// the recorded sound, unchanged
	samples []int16

	// the television sends audio in batches that do not start or end with a frame. but there are
	// always two samples for every scanline and the last sample in a batch is on the current
	// scanline. the offset is the number of samples minus two for every scanline, measured at the
	// end of the most recent batch. it is the same for every batch, give or take one sample
	offset      int
	offsetValid bool

	// the end of every frame since the recording started
	frames []frameEnd

	// the frame number of every picture sent to the video encoder, in order
	pictures []int

	// the sound has been fitted to the pictures and written to the wav file
	ended  bool
	fitted fitSummary
}

// audioRate returns the sample rate of the wav file for the specification
func audioRate(spec specification.Spec) int {
	return int(spec.HorizontalScanRate * float32(tiaaudio.SamplesPerScanline))
}

// samplesPerFrame returns the number of audio samples in a frame of the specification's length
func samplesPerFrame(spec specification.Spec) int {
	return spec.ScanlinesTotal * tiaaudio.SamplesPerScanline
}

func newAudio(tempAudioFilename string, spec specification.Spec) (*audio, error) {
	wavs, err := wavwriter.NewWavWriter(tempAudioFilename, audioRate(spec))
	if err != nil {
		return nil, err
	}
	return &audio{
		wavs:            wavs,
		samplesPerFrame: samplesPerFrame(spec),
	}, nil
}

// SetAudio implements the television.AudioMixer interface
func (au *audio) SetAudio(sig []signal.AudioSignalAttributes) error {
	au.crit.Lock()
	defer au.crit.Unlock()
	if au.ended {
		return nil
	}
	for _, s := range sig {
		au.samples = append(au.samples, mix.Mono(s.AudioChannel0, s.AudioChannel1))
	}
	au.offset = len(au.samples) - au.scanlines*tiaaudio.SamplesPerScanline
	au.offsetValid = true
	return nil
}

// NewScanline implements the television.ScanlineTrigger interface
func (au *audio) NewScanline(_ frameinfo.Current) error {
	au.scanlines++
	return nil
}

// NewFrame implements the television.FrameTrigger interface
func (au *audio) NewFrame(info frameinfo.Current) error {
	// a new frame is also the end of a scanline. the television calls NewFrame() instead of
	// NewScanline() in that case
	au.scanlines++

	au.crit.Lock()
	defer au.crit.Unlock()
	if au.ended {
		return nil
	}
	au.frames = append(au.frames, frameEnd{
		frameNum:    info.FrameNum,
		scanlines:   au.scanlines,
		offset:      au.offset,
		offsetValid: au.offsetValid,
	})
	return nil
}

// ready returns true if the numbered frame has ended since the recording started. the first picture
// should wait until the frame before it is ready, so that the start of its sound is known
func (au *audio) ready(frameNum int) bool {
	au.crit.Lock()
	defer au.crit.Unlock()
	return slices.ContainsFunc(au.frames, func(f frameEnd) bool {
		return f.frameNum == frameNum
	})
}

// addPicture notes that the picture for the numbered frame has been sent to the video encoder
func (au *audio) addPicture(frameNum int) {
	au.crit.Lock()
	defer au.crit.Unlock()
	au.pictures = append(au.pictures, frameNum)
}

// summary returns how the sound was fitted to the pictures. it is only valid after EndMixing()
func (au *audio) summary() fitSummary {
	au.crit.Lock()
	defer au.crit.Unlock()
	return au.fitted
}

// EndMixing implements the television.AudioMixer interface. the wav file is not written until the
// recording ends (see end()) because the last picture can arrive after the television has ended
func (au *audio) EndMixing() error {
	return nil
}

// end fits the sound to the pictures and writes the wav file
func (au *audio) end() error {
	au.crit.Lock()
	defer au.crit.Unlock()
	if au.ended {
		return nil
	}
	au.ended = true
	au.wavs.AddSamples(au.fit())
	au.samples = nil
	return au.wavs.EndMixing()
}

// Reset implements the television.AudioMixer interface
func (au *audio) Reset() {
	au.wavs.Reset()
}

// fit returns the sound for the video. every picture is given the sound for its frame, stretched or
// squeezed to samplesPerFrame samples, so the sound is the same length as the video and in step
// with it however many scanlines there are in each frame. a picture without any recorded sound is
// given silence
func (au *audio) fit() []int16 {
	au.fitted = fitSummary{pictures: len(au.pictures)}

	// the position in the recorded sound of the end of every frame. a frame that ended before any
	// samples were received takes the offset of the first frame that has one
	ends := make([]int, len(au.frames))
	offset := au.offset
	for i := len(au.frames) - 1; i >= 0; i-- {
		if au.frames[i].offsetValid {
			offset = au.frames[i].offset
		}
		// the last sample in a batch is the first or the second on its scanline. taking it to be
		// the second puts the end of the frame right to within a sample
		ends[i] = au.frames[i].scanlines*tiaaudio.SamplesPerScanline + offset - tiaaudio.SamplesPerScanline
	}

	out := make([]int16, 0, len(au.pictures)*au.samplesPerFrame)

	// pictures for consecutive frames are fitted together as a run. bounds are the positions of
	// the start of the run's first frame and the end of each frame in it
	var bounds []int
	fitRun := func() {
		if len(bounds) > 1 {
			sound, minLength, maxLength := fitSound(au.samples, bounds, au.samplesPerFrame)
			out = append(out, sound...)
			if au.fitted.maxLength == 0 {
				au.fitted.minLength = minLength
				au.fitted.maxLength = maxLength
			} else {
				au.fitted.minLength = min(au.fitted.minLength, minLength)
				au.fitted.maxLength = max(au.fitted.maxLength, maxLength)
			}
		}
		bounds = bounds[:0]
	}

	next := 0
	for _, p := range au.pictures {
		i := next
		for i < len(au.frames) && au.frames[i].frameNum != p {
			i++
		}
		if i >= len(au.frames) {
			fitRun()
			out = append(out, make([]int16, au.samplesPerFrame)...)
			au.fitted.noSound++
			continue
		}
		if i != next || len(bounds) == 0 {
			fitRun()
			if i > 0 {
				bounds = append(bounds, ends[i-1])
			} else {
				// the recording started part way through the first frame
				bounds = append(bounds, ends[i]-au.samplesPerFrame)
			}
		}
		bounds = append(bounds, ends[i])
		next = i + 1
	}
	fitRun()

	return out
}

// fitSound returns the sound for a run of consecutive frames, with samplesPerFrame samples for every
// frame. bounds are the positions in the sound of the start of the first frame and the end of every
// frame. the stretch follows the average length of the surrounding frames, so the start of a
// frame's sound can move by as much as the number of scanlines wobbles. the shortest and longest
// frames after averaging are also returned
func fitSound(sound []int16, bounds []int, samplesPerFrame int) ([]int16, float64, float64) {
	n := len(bounds) - 1

	// moving average of the bounds. the average is taken over fewer frames near the ends of the run
	// so that the first and last bounds stay where they are
	sum := make([]int64, len(bounds)+1)
	for i, b := range bounds {
		sum[i+1] = sum[i] + int64(b)
	}
	smooth := make([]float64, len(bounds))
	for i := range bounds {
		r := min(smoothFrames, i, n-i)
		smooth[i] = float64(sum[i+r+1]-sum[i-r]) / float64(2*r+1)
	}

	// frames that are all the specification's length give every sample unchanged
	out := make([]int16, 0, n*samplesPerFrame)
	minLength := math.Inf(1)
	maxLength := math.Inf(-1)
	for i := range n {
		start := smooth[i]
		length := smooth[i+1] - smooth[i]
		minLength = min(minLength, length)
		maxLength = max(maxLength, length)
		for s := range samplesPerFrame {
			out = append(out, soundAt(sound, start+length*float64(s)/float64(samplesPerFrame)))
		}
	}
	return out, minLength, maxLength
}

// soundAt returns the sound at a position in the recorded sound. a position between two samples is
// interpolated. a position before the first sample or after the last is given the nearest sample
func soundAt(sound []int16, pos float64) int16 {
	if len(sound) == 0 {
		return 0
	}
	i := int(math.Floor(pos))
	f := pos - float64(i)
	a := float64(sound[max(0, min(len(sound)-1, i))])
	if f == 0 {
		return int16(a)
	}
	b := float64(sound[max(0, min(len(sound)-1, i+1))])
	return int16(math.Round(a + (b-a)*f))
}
