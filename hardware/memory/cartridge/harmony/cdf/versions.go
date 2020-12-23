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

package cdf

// versions contains the information that can differ between CDF versions
type version struct {
	version byte

	// mappingID and description differ depending on the version
	submapping  string
	description string

	// the base index for the CDF registers. These values are indexes into the
	// data RAM.
	fetcherBase   uint32
	incrementBase uint32
	musicBase     uint32

	// which data fetcher is the amplitude fetcher differs by CFD version
	amplitudeRegister int

	// the significant bits of the most-significant byte of a fastjmp operand
	// must be masked appropriately. in the case of CDFJ a fast jump can be
	// triggered with either "4c 00 00" or "4c 01 00"
	fastJMPmask uint8

	// how we access the bits of the registers differ for different versions of
	// the CDF mapper
	fetcherShift   uint32
	incrementShift uint32

	// the DSPTR register is written one byte at a time from the 6507. How many
	// bytes are in the DSPTR depends on the size of the CDF ROM.
	fetcherMask uint32
}

func newVersion(v byte) (version, error) {
	r := version{
		version: v,
	}

	// different version of the CDF mapper have different addresses
	switch r.version {
	case 0x0: // cdf0
		r.submapping = "CDF0"
		r.description = "Harmony (CDF0)"
		r.fetcherBase = 0x06e0
		r.incrementBase = 0x0768
		r.musicBase = 0x07f0
		r.fastJMPmask = 0xff
		r.amplitudeRegister = 34
	case 0x4a: // cdfj
		r.submapping = "CDFJ"
		r.description = "Harmony (CDFJ)"
		r.fetcherBase = 0x0098
		r.incrementBase = 0x0124
		r.musicBase = 0x01b0
		r.fastJMPmask = 0xfe
		r.amplitudeRegister = 35
	default: // cdf1
		r.submapping = "CDF1"
		r.description = "Harmony (CDF1)"
		r.fetcherBase = 0x00a0
		r.incrementBase = 0x0128
		r.musicBase = 0x01b0
		r.fastJMPmask = 0xff
		r.amplitudeRegister = 34
	}

	r.fetcherShift = 20
	r.incrementShift = 12
	r.fetcherMask = 0xf0000000

	return r, nil
}
