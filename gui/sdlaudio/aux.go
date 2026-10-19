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
	"unsafe"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/veandco/go-sdl2/sdl"
)

type auxAudio struct {
	aux         bool
	auxId       sdl.AudioDeviceID
	auxOut      []int16
	auxSilence  []int16
	auxMaxQueue int
}

// OpenAuxAudio implements the television.AuxAudio interface
func (aud *Audio) OpenAuxAudio(sampleRate int, bufferLen int) error {
	if aud.aux {
		return nil
	}

	request := &sdl.AudioSpec{
		Freq:     int32(sampleRate),
		Format:   sdl.AUDIO_S16SYS,
		Channels: 1,
		Samples:  uint16(bufferLen),
	}

	// with no allowed changes SDL converts from the SpeakJet's own sample rate
	// to whatever the device needs
	auxId, err := sdl.OpenAudioDevice("", false, request, nil, 0)
	if err != nil {
		return fmt.Errorf("speakjet: %w", err)
	}
	sdl.PauseAudioDevice(auxId, false)

	aud.aux = true
	aud.auxId = auxId
	aud.auxOut = make([]int16, bufferLen)
	aud.auxSilence = make([]int16, bufferLen)

	// most audio allowed to queue on the device. if the emulation runs faster
	// than real time, chunks beyond this are not played (they are still
	// recorded)
	aud.auxMaxQueue = sampleRate / 4

	return nil
}

// CloseAuxAudio implements the television.AuxAudio interface
func (aud *Audio) CloseAuxAudio() error {
	if aud.aux {
		sdl.CloseAudioDevice(aud.auxId)
		aud.aux = false
	}
	return nil
}

// PushAuxAudio implements the television.AuxAudio interface
func (aud *Audio) PushAuxAudio(samples []float32, rate int) error {
	if !aud.aux {
		return nil
	}

	queue := func(samples []int16) {
		b := unsafe.Slice((*byte)(unsafe.Pointer(&samples[0])), len(samples)*2)
		if err := sdl.QueueAudio(aud.auxId, b); err != nil {
			logger.Logf(logger.Allow, "sdlaudio", "aux: %s", err.Error())
		}
	}

	queued := sdl.GetQueuedAudioSize(aud.auxId) / 2
	if queued == 0 {
		// silence queued ahead of speech that starts on an empty audio queue, so
		// that small variations in emulation speed don't starve the device
		for range 3 {
			queue(aud.auxSilence)
		}
	}

	// aux audio at full scale (-1 to 1) is as loud as the loudest TIA output
	const auxScale = 0x7fff

	if queued < uint32(aud.auxMaxQueue) {
		for i, v := range samples {
			aud.auxOut[i] = int16(v * auxScale)
		}
		b := unsafe.Slice((*byte)(unsafe.Pointer(&aud.auxOut[0])), len(samples)*2)
		if err := sdl.QueueAudio(aud.auxId, b); err != nil {
			logger.Logf(logger.Allow, "sdlaudio", "aux: %s", err.Error())
		}
	}

	return nil
}
