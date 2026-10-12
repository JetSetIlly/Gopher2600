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
	"sync"

	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	tiaaudio "github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

// the number of recent frames for which the audio sample count is kept. it only needs to cover the
// distance between the emulation and the GUI
const frameSamplesLen = 8

type frameSamples struct {
	frameNum int
	samples  int
	valid    bool
}

type audio struct {
	wavs    *wavwriter.WavWriter
	enabled bool

	// the number of samples sent to the wav file. this is the clock for the video
	samples int

	// the number of samples sent to the wav file by the end of each recent frame. written by
	// NewFrame() in the emulation goroutine and read by samplesAtFrame() in the GUI goroutine
	frameSamples     [frameSamplesLen]frameSamples
	frameSamplesLock sync.Mutex
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
		wavs:    wavs,
		enabled: true,
	}, nil
}

// SetAudio implements the television.AudioMixer interface
func (au *audio) SetAudio(sig []signal.AudioSignalAttributes) error {
	if !au.enabled {
		return nil
	}
	au.samples += len(sig)
	return au.wavs.SetAudio(sig)
}

// NewFrame implements the television.FrameTrigger interface
func (au *audio) NewFrame(info frameinfo.Current) error {
	if info.FrameNum < 0 {
		return nil
	}
	au.frameSamplesLock.Lock()
	defer au.frameSamplesLock.Unlock()
	au.frameSamples[info.FrameNum%frameSamplesLen] = frameSamples{
		frameNum: info.FrameNum,
		samples:  au.samples,
		valid:    true,
	}
	return nil
}

// samplesAtFrame returns the number of samples sent to the wav file by the end of the numbered
// frame. returns false if that frame is not one of the recent frames
func (au *audio) samplesAtFrame(frameNum int) (int, bool) {
	if frameNum < 0 {
		return 0, false
	}
	au.frameSamplesLock.Lock()
	defer au.frameSamplesLock.Unlock()
	fs := au.frameSamples[frameNum%frameSamplesLen]
	if !fs.valid || fs.frameNum != frameNum {
		return 0, false
	}
	return fs.samples, true
}

// EndMixing implements the television.AudioMixer interface
func (au *audio) EndMixing() error {
	return au.wavs.EndMixing()
}

// Reset implements the television.AudioMixer interface
func (au *audio) Reset() {
	au.wavs.Reset()
}
