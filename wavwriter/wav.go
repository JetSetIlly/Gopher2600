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

// Package wavwriter allows writing of audio data to disk as a WAV file.
package wavwriter

import (
	"fmt"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	tia "github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// WavWriter implements the television.AudioMixer interface
type WavWriter struct {
	filename string
	buffer   []int16
}

// New is the preferred method of initialisation for the WavWriter type
func NewWavWriter(filename string) (*WavWriter, error) {
	if !strings.HasSuffix(strings.ToLower(filename), ".wav") {
		filename = fmt.Sprintf("%s.wav", filename)
	}
	aw := &WavWriter{
		filename: filename,
		buffer:   make([]int16, 0),
	}
	return aw, nil
}

// SetAudio implements the television.AudioMixer interface.
func (aw *WavWriter) SetAudio(sig []signal.AudioSignalAttributes) error {
	for _, s := range sig {
		v0 := s.AudioChannel0
		v1 := s.AudioChannel1

		m := mix.Mono(v0, v1)
		aw.buffer = append(aw.buffer, m)
	}

	return nil
}

const numChannels = 1
const bitDepth = 16

// EndMixing implements the television.AudioMixer interface
func (aw *WavWriter) EndMixing() error {
	f, err := os.Create(aw.filename)
	if err != nil {
		return fmt.Errorf("wavwriter: %w", err)
	}
	defer f.Close()

	enc := wav.NewEncoder(f, tia.AverageSampleFreq, bitDepth, numChannels, 1)
	if enc == nil {
		return fmt.Errorf("wavwriter: bad parameters for wav encoding")
	}
	defer enc.Close()

	buf := audio.PCMBuffer{
		Format: &audio.Format{
			NumChannels: numChannels,
			SampleRate:  tia.AverageSampleFreq,
		},
		I16:            aw.buffer,
		DataType:       audio.DataTypeI16,
		SourceBitDepth: bitDepth,
	}

	err = enc.Write(buf.AsIntBuffer())
	if err != nil {
		return fmt.Errorf("wavwriter: %w", err)
	}

	return nil
}

// Reset implements the television.AudioMixer interface
func (aw *WavWriter) Reset() {
}
