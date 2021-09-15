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

// Package wavwriter allows writing of audio data to disk as a WAV file. Note
// that audio data is buffered in memory in its entirity, and written to disk
// on program end. It is therefore probably only suitable for testing purposes.
package wavwriter

import (
	"os"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/youpy/go-wav"
)

// WavWriter implements the television.AudioMixer interface.
type WavWriter struct {
	filename string
	buffer   []wav.Sample
}

// New is the preferred method of initialisation for the Audio2Wav type.
func New(filename string) (*WavWriter, error) {
	aw := &WavWriter{
		filename: filename,
		buffer:   make([]wav.Sample, 0),
	}

	return aw, nil
}

// SetAudio implements the television.AudioMixer interface.
func (aw *WavWriter) SetAudio(sig []signal.SignalAttributes) error {
	for _, s := range sig {
		if s&signal.AudioUpdate != signal.AudioUpdate {
			continue
		}

		v0 := uint8((s & signal.AudioChannel0) >> signal.AudioChannel0Shift)
		v1 := uint8((s & signal.AudioChannel1) >> signal.AudioChannel1Shift)
		v := mix.Mono(v0, v1)

		w := wav.Sample{}
		w.Values[0] = int(v >> 8)
		w.Values[1] = int(v)

		aw.buffer = append(aw.buffer, w)
	}

	return nil
}

// EndMixing implements the television.AudioMixer interface.
func (aw *WavWriter) EndMixing() (rerr error) {
	f, err := os.Create(aw.filename)
	if err != nil {
		return curated.Errorf("wavwriter: %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			rerr = curated.Errorf("wavwriter: %v", err)
		}
	}()

	// see audio commentary in sdlplay package for thinking around sample rates

	enc := wav.NewWriter(f, uint32(len(aw.buffer)), 1, uint32(audio.SampleFreq), 8)
	if enc == nil {
		return curated.Errorf("wavwriter: %v", "bad parameters for wav encoding")
	}

	logger.Logf("wavwriter", "writing audio to %s", aw.filename)
	enc.WriteSamples(aw.buffer)

	return nil
}

// Reset implements the television.AudioMixer interface.
func (aw *WavWriter) Reset() {
}
