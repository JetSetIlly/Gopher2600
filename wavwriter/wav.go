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
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// WavWriter implements the television.AudioMixer interface
type WavWriter struct {
	filename   string
	buffer     []int16
	sampleRate int
	ended      bool
}

// New is the preferred method of initialisation for the WavWriter type
func NewWavWriter(filename string, sampleRate int) (*WavWriter, error) {
	if !strings.HasSuffix(strings.ToLower(filename), ".wav") {
		filename = fmt.Sprintf("%s.wav", filename)
	}
	aw := &WavWriter{
		filename:   filename,
		buffer:     make([]int16, 0),
		sampleRate: sampleRate,
	}
	return aw, nil
}

// SetAudio implements the television.AudioMixer interface.
func (aw *WavWriter) SetAudio(sig []signal.AudioSignalAttributes) error {
	return aw.SetAudioWithAux(sig, nil)
}

// aux audio at full scale (-1 to 1) is as loud as the loudest TIA output
const auxScale = 0x7fff

// SetAudioWithAux is the same as SetAudio() but adds sound form an aux audio source (see
// television.AuxAudio). aux is at the same sample rate as sig and lines up with its start. it can
// be shorter than sig
func (aw *WavWriter) SetAudioWithAux(sig []signal.AudioSignalAttributes, aux []float32) error {
	if aw.ended {
		return nil
	}
	for i, s := range sig {
		v0 := s.AudioChannel0
		v1 := s.AudioChannel1

		m := int32(mix.Mono(v0, v1))
		if i < len(aux) {
			m += int32(aux[i] * auxScale)
		}
		aw.buffer = append(aw.buffer, int16(max(-0x8000, min(0x7fff, m))))
	}

	return nil
}

const numChannels = 1
const bitDepth = 16

// EndMixing implements the television.AudioMixer interface
func (aw *WavWriter) EndMixing() error {
	if aw.ended {
		return nil
	}
	aw.ended = true

	f, err := os.Create(aw.filename)
	if err != nil {
		return fmt.Errorf("wavwriter: %w", err)
	}
	defer f.Close()

	enc := wav.NewEncoder(f, aw.sampleRate, bitDepth, numChannels, 1)
	if enc == nil {
		return fmt.Errorf("wavwriter: bad parameters for wav encoding")
	}
	defer enc.Close()

	buf := audio.PCMBuffer{
		Format: &audio.Format{
			NumChannels: numChannels,
			SampleRate:  aw.sampleRate,
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
