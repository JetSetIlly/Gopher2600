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

package audio

// VolumeMix lookup table is created according to the information in the
// document, "TIA Sounding Off In The Digital Domain", by Chris Brenner.
//
// https://atariage.com/forums/topic/249865-tia-sounding-off-in-the-digital-domain/
var volumeMix [256]int16

func init() {
	var i int
	var r1, r2, ra, rb, rc, rd float32
	r1 = 1000.0
	ra = 1.0 / 3750.0
	rb = 1.0 / 7500.0
	rc = 1.0 / 15000.0
	rd = 1.0 / 30000.0
	volumeMix[0] = 0
	for i = 1; i < 256; i++ {
		r2 = 0.0
		if i&0x01 == 0x01 {
			r2 += rd
		}
		if i&0x02 == 0x02 {
			r2 += rc
		}
		if i&0x04 == 0x04 {
			r2 += rb
		}
		if i&0x08 == 0x08 {
			r2 += ra
		}
		if i&0x10 == 0x10 {
			r2 += rd
		}
		if i&0x20 == 0x20 {
			r2 += rc
		}
		if i&0x40 == 0x40 {
			r2 += rb
		}
		if i&0x80 == 0x80 {
			r2 += ra
		}
		r2 = 1.0 / r2
		volumeMix[i] = int16(32768.0*(1.0-r2/(r1+r2)) + 0.5)
	}
}
