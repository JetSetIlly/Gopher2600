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

// Package mix is used to combine two distinct sound sources into either a mono
// or stereo signal.
//
// The mono mix is created according to the information in the document, "TIA
// Sounding Off In The Digital Domain", by Chris Brenner. Announcment link
// below:
//
// https://atariage.com/forums/topic/249865-tia-sounding-off-in-the-digital-domain/
//
// The exact implementation here is an optimisation of that work, as found by
// Thomas Jentzsch (mentioned in the link above)
//
// Both 6502.ts and Stella source was used as reference. Both projects are
// exactly equivalent.
//
//	  6502.ts (published under the MIT licence)
//			https://github.com/6502ts/6502.ts/blob/6f923e5fe693b82a2448ffac1f85aea9693cacff/src/machine/stella/tia/PCMAudio.ts
//			https://github.com/6502ts/6502.ts/blob/6f923e5fe693b82a2448ffac1f85aea9693cacff/src/machine/stella/tia/PCMChannel.ts
//
//	 Stella (published under the GNU GPL v2.0 licence)
//			https://github.com/stella-emu/stella/blob/e6af23d6c12893dd17711002971087f28f87c31f/src/emucore/tia/Audio.cxx
//			https://github.com/stella-emu/stella/blob/e6af23d6c12893dd17711002971087f28f87c31f/src/emucore/tia/AudioChannel.cxx
package mix

const maxVolume = 0x1e

var mono [maxVolume + 1]int16

// Mono returns a single volume value.
func Mono(channel0 uint8, channel1 uint8) int16 {
	return mono[int16(channel0+channel1)] >> 1
}

// Stereo return a pair of volume values.
func Stereo(channel0 uint8, channel1 uint8) (int16, int16) {
	return Mono(channel0, 0), Mono(0, channel1)
}

func init() {
	for vol := 0; vol < len(mono); vol++ {
		mono[vol] = int16(0x7fff * float32(vol) / float32(maxVolume) * (30 + 1*float32(maxVolume)) / (30 + 1*float32(vol)))
	}
}
