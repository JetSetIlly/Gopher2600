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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package wavwriter allows writing of audio data to disk as a WAV file.
package wavwriter

import (
	"os"

	"github.com/jetsetilly/gopher2600/errors"
	tiaAudio "github.com/jetsetilly/gopher2600/hardware/tia/audio"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// WavWriter implements the television.AudioMixer interface
type WavWriter struct {
	filename string
	buffer   []int8
}

// New is the preferred method of initialisation for the Audio2Wav type
func New(filename string) (*WavWriter, error) {
	aw := &WavWriter{
		filename: filename,
		buffer:   make([]int8, 0, 0),
	}

	return aw, nil
}

// SetAudio implements the television.AudioMixer interface
func (aw *WavWriter) SetAudio(audioData uint8) error {
	// bring audioData into the correct range
	aw.buffer = append(aw.buffer, int8(int16(audioData)-127))
	return nil
}

// FlushAudio implements the television.AudioMixer interface
func (aw *WavWriter) FlushAudio() error {
	return nil
}

// PauseAudio implements the television.AudioMixer interface
func (aw *WavWriter) PauseAudio(pause bool) error {
	return nil
}

// EndMixing implements the television.AudioMixer interface
func (aw *WavWriter) EndMixing() error {
	err := aw.FlushAudio()
	if err != nil {
		return errors.New(errors.WavWriter, err)
	}

	f, err := os.Create(aw.filename)
	if err != nil {
		return errors.New(errors.WavWriter, err)
	}
	defer f.Close()

	// see audio commentary in sdlplay package for thinking around sample rates

	enc := wav.NewEncoder(f, tiaAudio.SampleFreq, 8, 1, 1)
	if enc == nil {
		return errors.New(errors.WavWriter, "bad parameters for wav encoding")
	}
	defer enc.Close()

	buf := audio.PCMBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  tiaAudio.SampleFreq,
		},
		I8:             aw.buffer,
		DataType:       audio.DataTypeI8,
		SourceBitDepth: 8,
	}

	err = enc.Write(buf.AsIntBuffer())
	if err != nil {
		return errors.New(errors.WavWriter, err)
	}

	return nil
}
