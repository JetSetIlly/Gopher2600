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

package digest

import (
	"crypto/sha1"
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/television"
)

// the length of the buffer we're using isn't really important. that said, it
// needs to be at least sha1.Size bytes in length.
const audioBufferLength = 1024 + sha1.Size

// to allow us to create digests on audio streams longer than
// audioBufferLength, we'll stuff the previous digest value into the first part
// of the buffer array and make sure we include it when we create the next
// digest value.
const audioBufferStart = sha1.Size

// Audio is an implementation of the television.AudioMixer interface with an
// embedded television for convenience. It periodically generates a SHA-1 value
// of the audio stream.
//
// Note that the use of SHA-1 is fine for this application because this is not a
// cryptographic task.
type Audio struct {
	television.Television
	digest   [sha1.Size]byte
	buffer   []uint8
	bufferCt int
}

// NewAudio is the preferred method of initialisation for the Audio2Wav type.
func NewAudio(tv television.Television) (*Audio, error) {
	dig := &Audio{Television: tv}

	// register ourselves as a television.AudioMixer
	dig.AddAudioMixer(dig)

	// create buffer
	dig.buffer = make([]uint8, audioBufferLength)
	dig.bufferCt = audioBufferStart

	return dig, nil
}

// Hash implements digest.Digest interface.
func (dig Audio) Hash() string {
	return fmt.Sprintf("%x", dig.digest)
}

// ResetDigest implements digest.Digest interface.
func (dig *Audio) ResetDigest() {
	for i := range dig.digest {
		dig.digest[i] = 0
	}
}

// SetAudio implements the television.AudioMixer interface.
func (dig *Audio) SetAudio(audioData uint8) error {
	dig.buffer[dig.bufferCt] = audioData

	dig.bufferCt++

	if dig.bufferCt >= audioBufferLength {
		return dig.flushAudio()
	}

	return nil
}

func (dig *Audio) flushAudio() error {
	dig.digest = sha1.Sum(dig.buffer)
	n := copy(dig.buffer, dig.digest[:])
	if n != len(dig.digest) {
		return curated.Errorf("digest: audio: digest error while flushing audio stream")
	}
	dig.bufferCt = audioBufferStart
	return nil
}

// EndMixing implements the television.AudioMixer interface.
func (dig *Audio) EndMixing() error {
	return nil
}
