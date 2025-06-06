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

	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

// Video is an implementation of the television.PixelRenderer interface with an
// embedded television for convenience. It generates a SHA-1 value of the image
// every frame. it does not display the image anywhere.
//
// Note that the use of SHA-1 is fine for this application because this is not
// a cryptographic task.
type Video struct {
	*television.Television
	spec     specification.Spec
	digest   [sha1.Size]byte
	pixels   []byte
	frameNum int
}

// NewVideo initialises a new instance of DigestTV.
func NewVideo(tv *television.Television) (*Video, error) {
	// set up digest tv
	dig := &Video{
		Television: tv,
		spec:       specification.SpecNTSC,
	}

	// register ourselves as a television.Renderer
	dig.AddPixelRenderer(dig)

	// length of pixels array contains enough room for the previous frames digest value
	l := len(dig.digest)
	l += specification.AbsoluteMaxClks

	// allocate enough pixels for entire frame
	dig.pixels = make([]byte, l)

	return dig, nil
}

// Hash implements digest.Digest interface.
func (dig Video) Hash() string {
	return fmt.Sprintf("%x", dig.digest)
}

// ResetDigest implements digest.Digest interface.
func (dig *Video) ResetDigest() {
	for i := range dig.digest {
		dig.digest[i] = 0
	}
}

// Resize implements television.PixelRenderer interface
//
// In this implementation we only handle specification changes. This means the
// digest is immune from changes to the frame resizing method used by the
// television implementation. Changes to how the specification is flipped might
// cause comparison failures however.
func (dig *Video) Resize(frameInfo frameinfo.Current) error {
	dig.spec = frameInfo.Spec
	return nil
}

// NewFrame implements television.PixelRenderer interface.
func (dig *Video) NewFrame(_ frameinfo.Current) error {
	// chain fingerprints by copying the value of the last fingerprint
	// to the head of the video data
	n := copy(dig.pixels, dig.digest[:])
	if n != len(dig.digest) {
		return fmt.Errorf("digest: video: digest error during new frame")
	}
	dig.digest = sha1.Sum(dig.pixels)
	dig.frameNum++
	return nil
}

// NewScanline implements television.PixelRenderer interface.
func (dig *Video) NewScanline(scanline int) error {
	return nil
}

// SetPixels implements television.PixelRenderer interface.
func (dig *Video) SetPixels(sig []signal.SignalAttributes, _ int) error {
	// offset always starts after the digest leader
	offset := len(dig.digest)

	for _, s := range sig {
		// ignore invalid signals. this has a consequence that shrunken screen
		// sizes will have pixel values left over from previous frames. but for
		// the purposes of the digest this is okay
		if s.Index == signal.NoSignal {
			continue
		}
		dig.pixels[offset] = byte(s.Color)
		offset++
	}
	return nil
}

// Reset implements television.PixelRenderer interface.
func (dig *Video) Reset() {
}

// EndRendering implements television.PixelRenderer interface.
func (dig *Video) EndRendering() error {
	return nil
}
