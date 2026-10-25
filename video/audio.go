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
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	tiaaudio "github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

type audio struct {
	wavs    *wavwriter.WavWriter
	enabled bool
	clip    int
}

func newAudio(tempAudioFilename string, frameInfo frameinfo.Current) (*audio, error) {
	rate := frameInfo.Spec.HorizontalScanRate * float32(tiaaudio.SamplesPerScanline)
	wavs, err := wavwriter.NewWavWriter(tempAudioFilename, rate)
	if err != nil {
		return nil, err
	}
	clip := tiaaudio.SamplesPerScanline * frameInfo.TotalScanlines
	wavs.SetClip(clip)
	return &audio{
		wavs:    wavs,
		enabled: true,
		clip:    clip,
	}, nil
}

// SetAuxAudio implements the television.AuxAudio interface
func (au *audio) SetAuxAudio(samples []float32, rate int) error {
	if !au.enabled {
		return nil
	}
	return au.wavs.SetAuxAudio(samples, rate)
}

// SetAudio implements the television.AudioMixer interface
func (au *audio) SetAudio(sig []signal.AudioSignalAttributes) error {
	if !au.enabled {
		return nil
	}
	return au.wavs.SetAudio(sig)
}

func (au *audio) NewFrame(frameInfo frameinfo.Current) error {
	return au.wavs.NewFrame(frameInfo)
}

// EndMixing implements the television.AudioMixer interface
func (au *audio) EndMixing() error {
	return au.wavs.EndMixing()
}

// Reset implements the television.AudioMixer interface
func (au *audio) Reset() {
	au.wavs.Reset()
}
