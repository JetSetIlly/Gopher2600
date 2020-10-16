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

// Package digest is used to create mathematical hashes of VCS output. The
// two implementations of the Digest interface also implement the
// television.PixelRenderer and television.AudioMixer interfaces.
//
// The digest.Video type is used to capture video output while digest.Audio is
// used to capture audio output.
//
// The hashes produced by these types are used from regression tests and for
// verification of playback scripts.
package digest

// Digest implementations compute a mathematical hash, retreivable with the
// Hash() function.
type Digest interface {
	Hash() string
	ResetDigest()
}
