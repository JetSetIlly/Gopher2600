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

package sdlaudio

import (
	"fmt"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/tia/audio"

	"github.com/veandco/go-sdl2/sdl"
)

// the buffer length is important to get right. unfortunately, there's no
// special way (that I know of) that can tells us what the ideal value is. we
// don't want it to be long because we can introduce unnecessary lag between
// the audio and video signal; by the same token we don't want it too short because
// we will end up calling FlushAudio() too often - FlushAudio() is a
// computationally expensive function.
//
// the following value has been discovered through trial and error. the precise
// value is not critical.
const bufferLength = 512

// Audio outputs sound using SDL
type Audio struct {
	id   sdl.AudioDeviceID
	spec sdl.AudioSpec

	// we keep two buffers which we swap after every flush. the other buffer
	// can then be used to repeat and to fill in the gaps in the audio. see
	// repeatAudio()
	buffer   *[]uint8
	other    *[]uint8
	bufferA  []uint8
	bufferB  []uint8
	bufferCt int

	// some ROMs do not output 0 as the silence value. silence is technically
	// caused by constant unchanging value so this shouldn't be a problem. the
	// problem is caused when there is an audio buffer underflow and the sound
	// device flips to the real silence value - this causes a audible click.
	//
	// to mitigate this we try to detect what the silence value is by counting
	// the number of unchanging values
	detectedSilenceValue uint8
	lastAudioData        uint8
	countAudioData       int

	isBufferEmpty chan bool
}

// the number of consecutive cycles for an audio signal to be considered the
// new silence value
const audioDataSilenceThreshold = 10000

// NewAudio is the preferred method of initialisatoin for the Audio Type
func NewAudio() (*Audio, error) {
	aud := &Audio{
		isBufferEmpty: make(chan bool),
	}

	aud.bufferA = make([]uint8, bufferLength)
	aud.bufferB = make([]uint8, bufferLength)
	aud.buffer = &aud.bufferA
	aud.other = &aud.bufferB

	spec := &sdl.AudioSpec{
		// TODO: reduce playback frequency according to actual speed of emulation
		// Freq: int32(math.Floor(float64(audio.SampleFreq) * 0.90)),

		Freq:     audio.SampleFreq,
		Format:   sdl.AUDIO_U8,
		Channels: 1,
		Samples:  uint16(bufferLength),
	}

	var err error
	var actualSpec sdl.AudioSpec

	aud.id, err = sdl.OpenAudioDevice("", false, spec, &actualSpec, 0)
	if err != nil {
		return nil, err
	}

	aud.spec = actualSpec
	aud.detectedSilenceValue = aud.spec.Silence

	// fill buffers with silence
	for i, _ := range aud.bufferA {
		aud.bufferA[i] = aud.spec.Silence
	}
	for i, _ := range aud.bufferB {
		aud.bufferB[i] = aud.spec.Silence
	}

	go func() {
		rate := float64(bufferLength) / audio.SampleFreq
		dur, _ := time.ParseDuration(fmt.Sprintf("%fs", rate))
		tck := time.NewTicker(dur)
		for {
			_ = <-tck.C
			aud.isBufferEmpty <- true
		}
	}()

	sdl.PauseAudioDevice(aud.id, false)

	return aud, nil
}

// SetAudio implements the television.AudioMixer interface
func (aud *Audio) SetAudio(audioData uint8) error {
	select {
	case <-aud.isBufferEmpty:
		_ = aud.repeatAudio()
	default:
	}

	// silence detector
	if audioData == aud.lastAudioData && aud.countAudioData <= audioDataSilenceThreshold {
		aud.countAudioData++
		if aud.countAudioData > audioDataSilenceThreshold {
			aud.detectedSilenceValue = audioData
		}
	} else {
		aud.lastAudioData = audioData
		aud.countAudioData = 0
	}

	// never allow sound buffer to "output" silence - some sound devices take
	// an appreciable amount of time to move from silence to non-silence
	if audioData == aud.detectedSilenceValue {
		(*aud.buffer)[aud.bufferCt] = aud.spec.Silence
	} else {
		(*aud.buffer)[aud.bufferCt] = audioData + aud.spec.Silence
	}
	aud.bufferCt++

	if aud.bufferCt >= len(*aud.buffer) {
		return aud.flushAudio()
	}

	return nil
}

func (aud *Audio) flushAudio() error {
	sdl.ClearQueuedAudio(aud.id)
	err := sdl.QueueAudio(aud.id, *aud.buffer)
	if err != nil {
		return err
	}
	aud.bufferCt = 0
	aud.other = aud.buffer
	if aud.buffer == &aud.bufferA {
		aud.buffer = &aud.bufferA
	} else {
		aud.buffer = &aud.bufferB
	}

	return nil
}

func (aud *Audio) repeatAudio() error {
	return sdl.QueueAudio(aud.id, *aud.other)
}

// EndMixing implements the television.AudioMixer interface
func (aud *Audio) EndMixing() error {
	defer sdl.CloseAudioDevice(aud.id)
	return aud.flushAudio()
}
