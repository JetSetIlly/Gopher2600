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
	"math"
	"os"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

// WavWriter implements the television.AudioMixer interface
type WavWriter struct {
	filename   string
	buffer     []int16
	sampleRate float32
	ended      bool

	// audio signals from the television before they've been converted to samples for the buffer
	sig     []signal.AudioSignalAttributes
	sigClip int

	// sound from outside the TIA, resampled to the recording's sample rate and
	// waiting to be mixed into the next frames
	auxSamples []float32
	auxPos     float64
	auxPrev    float32
}

// New is the preferred method of initialisation for the WavWriter type
func NewWavWriter(filename string, sampleRate float32) (*WavWriter, error) {
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

func (aw *WavWriter) SetClip(clip int) {
	aw.sigClip = clip
}

// SampleRate returns the sample rate the WavWriter instance was created with
func (aw *WavWriter) SampleRate() float32 {
	return aw.sampleRate
}

// SetAudio implements the television.AudioMixer interface.
func (aw *WavWriter) SetAudio(sig []signal.AudioSignalAttributes) error {
	if aw.ended {
		return nil
	}
	aw.sig = append(aw.sig, sig...)
	return nil
}

// NewFrame implements the television.FrameTrigger interface
func (aw *WavWriter) NewFrame(frameInfo frameinfo.Current) error {
	if aw.ended {
		return nil
	}

	sig := aw.sig[:]
	if aw.sigClip > 0 {
		if len(sig) > aw.sigClip {
			sig = sig[:aw.sigClip]
		} else if len(sig) < aw.sigClip {
			sig = append(sig, make([]signal.AudioSignalAttributes, aw.sigClip-len(sig))...)
		}
	}
	aw.sig = aw.sig[:0]

	// the aux audio produced during this frame
	n := min(len(aw.auxSamples), len(sig))
	err := aw.setAudioWithAux(sig, aw.auxSamples[:n])
	aw.auxSamples = aw.auxSamples[:copy(aw.auxSamples, aw.auxSamples[n:])]

	return err
}

// SetAuxAudio implements the television.AuxAudio interface
func (aw *WavWriter) SetAuxAudio(samples []float32, rate int) error {
	if aw.ended {
		return nil
	}

	// linear interpolation from the source rate to the recording rate. auxPos
	// is the position of the next output sample between the previous source
	// sample (0) and the next one (1)
	step := float64(rate) / float64(aw.sampleRate)
	for _, s := range samples {
		for aw.auxPos < 1 {
			aw.auxSamples = append(aw.auxSamples, aw.auxPrev+(s-aw.auxPrev)*float32(aw.auxPos))
			aw.auxPos += step
		}
		aw.auxPos--
		aw.auxPrev = s
	}

	// the source and the television can drift apart by a little. don't let a backlog build up
	if aw.sigClip > 0 {
		if len(aw.auxSamples) > 4*aw.sigClip {
			aw.auxSamples = aw.auxSamples[len(aw.auxSamples)-4*aw.sigClip:]
		}
	}

	return nil
}

func (aw *WavWriter) setAudioWithAux(sig []signal.AudioSignalAttributes, aux []float32) error {
	for i, s := range sig {
		v0 := s.AudioChannel0
		v1 := s.AudioChannel1

		m := int32(mix.Mono(v0, v1))
		if i < len(aux) {
			m += int32(aux[i] * math.MaxInt16)
		}
		aw.buffer = append(aw.buffer, int16(max(math.MinInt16, min(math.MaxInt16, m))))
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

	enc := wav.NewEncoder(f, int(aw.sampleRate), bitDepth, numChannels, 1)
	if enc == nil {
		return fmt.Errorf("wavwriter: bad parameters for wav encoding")
	}
	defer enc.Close()

	buf := audio.PCMBuffer{
		Format: &audio.Format{
			NumChannels: numChannels,
			SampleRate:  int(aw.sampleRate),
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
