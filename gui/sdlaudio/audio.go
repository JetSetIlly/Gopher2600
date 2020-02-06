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
	"gopher2600/hardware/tia/audio"

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
const bufferLength = 256

// Audio outputs sound using SDL
type Audio struct {
	id       sdl.AudioDeviceID
	spec     sdl.AudioSpec
	buffer   []uint8
	bufferCt int
}

// NewAudio is the preferred method of initialisatoin for the Audio Type
func NewAudio() (*Audio, error) {
	aud := &Audio{}

	aud.buffer = make([]uint8, bufferLength)

	spec := &sdl.AudioSpec{
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

	// make sure audio device is unpaused on startup
	sdl.PauseAudioDevice(aud.id, false)

	return aud, nil
}

// SetAudio implements the television.AudioMixer interface
func (aud *Audio) SetAudio(audioData uint8) error {
	if aud.bufferCt >= len(aud.buffer) {
		return aud.FlushAudio()
	}

	// never allow sound buffer to "output" silence - some sound devices take
	// an appreciable amount of time to move from silence to non-silence
	//
	// if audioData == 0 {
	// 	aud.buffer[aud.bufferCt] = aud.spec.Silence + 1
	// } else {
	// 	aud.buffer[aud.bufferCt] = audioData + aud.spec.Silence
	// }

	aud.buffer[aud.bufferCt] = audioData + aud.spec.Silence
	aud.bufferCt++

	return nil
}

// FlushAudio implements the television.AudioMixer interface
func (aud *Audio) FlushAudio() error {
	err := sdl.QueueAudio(aud.id, aud.buffer)
	if err != nil {
		return err
	}
	aud.bufferCt = 0

	return nil
}

// PauseAudio implements the television.AudioMixer interface
func (aud *Audio) PauseAudio(pause bool) error {
	sdl.PauseAudioDevice(aud.id, pause)
	return nil
}

// EndMixing implements the television.AudioMixer interface
func (aud *Audio) EndMixing() error {
	defer sdl.CloseAudioDevice(aud.id)
	return aud.FlushAudio()
}
