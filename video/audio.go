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
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	tiaaudio "github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

type audio struct {
	wavs *wavwriter.WavWriter
}

func newAudio(tempAudioFilename string, spec specification.Spec) (*audio, error) {
	rate := spec.HorizontalScanRate * float32(tiaaudio.SamplesPerScanline)
	wavs, err := wavwriter.NewWavWriter(tempAudioFilename, int(rate))
	if err != nil {
		return nil, err
	}
	return &audio{
		wavs: wavs,
	}, nil
}

// SetAudio implements the television.AudioMixer interface
func (au *audio) SetAudio(sig []signal.AudioSignalAttributes) error {
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
