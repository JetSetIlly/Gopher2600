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

package sdlaudio

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/logger"

	"github.com/veandco/go-sdl2/sdl"
)

// the buffer length is important to get right. unfortunately, there's no
// special way (that I know of) that can tells us what the ideal value is
//
// the bufferLegnth value is the maximum size of the buffer. once the buffer is
// full the audio will be queued.
const bufferLength = 1024

// if the audio queue is ever less than minQueueLength then the buffer
// will be pushed to the queue immediately.
const minQueueLength = 256

// if audio queue is ever less than critQueueLength the the buffer is pushed to
// the queue but the buffer is not reset.
const critQueueLength = 64

// if queued audio ever exceeds this value then clip the audio.
const maxQueueLength = 8192

// Audio outputs sound using SDL.
type Audio struct {
	id   sdl.AudioDeviceID
	spec sdl.AudioSpec

	buffer   []uint8
	bufferCt int
}

// NewAudio is the preferred method of initialisatoin for the Audio Type.
func NewAudio() (*Audio, error) {
	aud := &Audio{
		buffer: make([]uint8, bufferLength),
	}

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

	logger.Log("sdl audio", fmt.Sprintf("frequency: %d samples/sec", aud.spec.Freq))
	logger.Log("sdl audio", fmt.Sprintf("format: %d", aud.spec.Format))
	logger.Log("sdl audio", fmt.Sprintf("channels: %d", aud.spec.Channels))
	logger.Log("sdl audio", fmt.Sprintf("buffer size: %d samples", aud.spec.Samples))

	// fill buffers with silence
	for i := range aud.buffer {
		aud.buffer[i] = aud.spec.Silence
	}

	sdl.PauseAudioDevice(aud.id, false)

	return aud, nil
}

// SetAudio implements the television.AudioMixer interface.
func (aud *Audio) SetAudio(audioData uint8) error {
	aud.buffer[aud.bufferCt] = audioData + aud.spec.Silence
	aud.bufferCt++

	if aud.bufferCt >= len(aud.buffer) {
		// if buffer is full then queue audio unconditionally
		err := sdl.QueueAudio(aud.id, aud.buffer)
		if err != nil {
			return err
		}
		aud.bufferCt = 0
	} else {
		remaining := int(sdl.GetQueuedAudioSize(aud.id))

		if remaining < critQueueLength {
			// if we're running short of bits in the queue the queue what we have
			// in the buffer and NOT clearing the buffer
			//
			// condition valid when the frame rate is SIGNIFICANTLY LESS than 50/60fps
			err := sdl.QueueAudio(aud.id, aud.buffer)
			if err != nil {
				return err
			}
		} else if remaining < minQueueLength && aud.bufferCt > 10 {
			// if we're running short of bits in the queue the queue what we have
			// in the buffer.
			//
			// condition valid when the frame rate is LESS than 50/60fps
			//
			// the additional condition makes sure we're not queueing a slice
			// that is too short. SDL has been known to hang with short audio
			// queues
			err := sdl.QueueAudio(aud.id, aud.buffer[:aud.bufferCt-1])
			if err != nil {
				return err
			}
			aud.bufferCt = 0
		} else if remaining > maxQueueLength {
			// if length of SDL audio queue is getting too long then clear it
			//
			// condition valid when the frame rate is SIGNIFICANTLY MORE than 50/60fps
			//
			// if we don't do this the video will get ahead of the audio (ie. the audio
			// will lag)
			//
			// this is a brute force approach but it'll do for now
			sdl.ClearQueuedAudio(aud.id)
		}
	}

	return nil
}

// EndMixing implements the television.AudioMixer interface.
func (aud *Audio) EndMixing() error {
	sdl.CloseAudioDevice(aud.id)
	return nil
}
