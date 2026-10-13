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
	sig     []signal.AudioSignalAttributes
}

func newAudio(tempAudioFilename string, frameInfo frameinfo.Current) (*audio, error) {
	rate := frameInfo.Spec.HorizontalScanRate * float32(tiaaudio.SamplesPerScanline)
	wavs, err := wavwriter.NewWavWriter(tempAudioFilename, int(rate))
	if err != nil {
		return nil, err
	}
	return &audio{
		wavs:    wavs,
		enabled: true,
		clip:    tiaaudio.SamplesPerScanline * frameInfo.TotalScanlines,
	}, nil
}

// SetAudio implements the television.AudioMixer interface
func (au *audio) SetAudio(sig []signal.AudioSignalAttributes) error {
	if !au.enabled {
		return nil
	}
	au.sig = append(au.sig, sig...)
	return nil
}

func (au *audio) NewFrame(frameInfo frameinfo.Current) error {
	sig := au.sig[:]
	if len(sig) > au.clip {
		sig = sig[:au.clip]
	} else if len(sig) < au.clip {
		sig = append(sig, make([]signal.AudioSignalAttributes, au.clip-len(sig))...)
	}
	au.sig = au.sig[:0]

	return au.wavs.SetAudio(sig)
}

// EndMixing implements the television.AudioMixer interface
func (au *audio) EndMixing() error {
	return au.wavs.EndMixing()
}

// Reset implements the television.AudioMixer interface
func (au *audio) Reset() {
	au.wavs.Reset()
}
