package digest

import (
	"crypto/sha1"
	"fmt"
	"gopher2600/errors"
)

// the length of the buffer we're using isn't really important. that said, it
// needs to be at least sha1.Size bytes in length
const audioBufferLength = 1024 + sha1.Size

// to allow us to create digests on audio streams longer than
// audioBufferLength, we'll stuff the previous digest value into the first part
// of the buffer array and make sure we include it when we create the next
// digest value
const audioBufferStart = sha1.Size

// Audio implemented the television.AudioMixer interface
type Audio struct {
	digest   [sha1.Size]byte
	buffer   []uint8
	bufferCt int
}

// NewAudio is the preferred method of initialisation for the Audio2Wav type
func NewAudio(filename string) (*Audio, error) {
	dig := &Audio{}
	dig.buffer = make([]uint8, audioBufferLength)
	dig.bufferCt = audioBufferStart
	return dig, nil
}

func (dig Audio) String() string {
	return fmt.Sprintf("%x", dig.digest)
}

// ResetDigest resets the current digest value to 0
func (dig *Audio) ResetDigest() {
	for i := range dig.digest {
		dig.digest[i] = 0
	}
}

// SetAudio implements the television.AudioMixer interface
func (dig *Audio) SetAudio(audioData uint8) error {
	dig.buffer[dig.bufferCt] = audioData

	dig.bufferCt++

	if dig.bufferCt >= audioBufferLength {
		return dig.FlushAudio()
	}

	return nil
}

// FlushAudio implements the television.AudioMixer interface
func (dig *Audio) FlushAudio() error {
	dig.digest = sha1.Sum(dig.buffer)
	n := copy(dig.buffer, dig.digest[:])
	if n != len(dig.digest) {
		return errors.New(errors.AudioDigest, fmt.Sprintf("digest error while flushing audio stream"))
	}
	dig.bufferCt = audioBufferStart
	return nil
}

// PauseAudio implements the television.AudioMixer interface
func (dig *Audio) PauseAudio(pause bool) error {
	return nil
}

// EndMixing implements the television.AudioMixer interface
func (dig *Audio) EndMixing() error {
	return nil
}
