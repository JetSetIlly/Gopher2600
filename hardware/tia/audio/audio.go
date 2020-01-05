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

package audio

import (
	"math/rand"
	"strings"
)

// SampleFreq represents the number of samples generated per second. This is
// the 30Khz reference frequency desribed in the Stella Programmer's Guide.
const SampleFreq = 31403

// the Atari 2600 has two independent sound generators. these will be mixed
// into one value by the Mix() function
const numChannels = 2

// Audio is the implementation of the TIA audio sub-system, using Ron Fries'
// method. Reference source code here:
//
// https://raw.githubusercontent.com/alekmaul/stella/master/emucore/TIASound.c
type Audio struct {
	// clock114 is so called because of the observation that the 30Khz
	// reference frequency described in the Stella Programmer's Guide is
	// generated from the 3.58Mhz clock divided by 114, giving a sample
	// frequency of 31403Hz or 31Khz - close enought to the 30Khz referency
	// frequency we need.  Ron Fries' talks about this in  his original
	// documentation for TIASound.c
	//
	// see the Mix() function to see how it is used
	clock114 int

	poly4bit [15]uint8
	poly5bit [31]uint8
	poly9bit [511]uint16
	div31    [31]uint8

	// From the "Stella Programmer's Guide":
	//
	// "There are two audio circuits for generating sound. They are identical but
	// completely independent and can be operated simultaneously [...]"
	channel0 channel
	channel1 channel
}

func (au *Audio) String() string {
	s := strings.Builder{}
	s.WriteString("ch0: ")
	s.WriteString(au.channel0.String())
	s.WriteString("  ch1: ")
	s.WriteString(au.channel1.String())
	return s.String()
}

// NewAudio is the preferred method of initialisation for the Video structure
func NewAudio() *Audio {
	au := &Audio{}
	au.channel0.au = au
	au.channel1.au = au

	// from TIASound.c:
	//
	// "Initialze the bit patterns for the polynomials.  The 4bit and 5bit patterns
	// are the identical ones used in the tia chip.  Though the patterns could be
	// packed with 8 bits per byte, using only a single bit per byte keeps the math
	// simple, which is important for efficient processing."
	au.poly4bit = [15]uint8{1, 1, 0, 1, 1, 1, 0, 0, 0, 0, 1, 0, 1, 0, 0}
	au.poly5bit = [31]uint8{0, 0, 1, 0, 1, 1, 0, 0, 1, 1, 1, 1, 1, 0, 0,
		0, 1, 1, 0, 1, 1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 1}

	// from TIASound.c (referring to 9 bit polynomial table):
	//
	// "Rather than have a table with 511 entries, I use a random number
	// generator."
	for i := 0; i < len(au.poly9bit); i++ {
		au.poly9bit[i] = uint16(rand.Int() & 0x01)
	}

	// from TIASound.c:
	//
	// "I've treated the 'Div by 31' counter as another polynomial because of the
	// way it operates.  It does not have a 50% duty cycle, but instead has a 13:18
	// ratio (of course, 13+18 = 31).  This could also be implemented by using
	// counters."
	au.div31 = [31]uint8{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	return au
}

// Mix the two VCS audio channels, returning a boolean indicating whether the
// sound has been updated and a single value representing the mixed volume
func (au *Audio) Mix() (bool, uint8) {
	// the reference frequency for all sound produced by the TIA is 30Khz. this
	// is the 3.58Mhz clock, which the TIA operates at, divided by 114 (see
	// declaration). Mix() is called every video cycle and we return
	// immediately except on the 114th tick, whereupon we process the current
	// audio registers and mix the two signals
	au.clock114++
	if au.clock114 < 114 {
		return false, 0
	}

	// reset clock114
	au.clock114 = 0

	// process each channel before mixing
	au.channel0.tick()
	au.channel1.tick()

	// mix channels: deciding the combined output volume for the two channels
	// is not as straight-forward and is it first seems. what we have here is
	// the naive implementation, simply adding the two volume values together
	// (we're not even taking an average).
	//
	// because the 2600 sound generator is an analogue circuit however, there
	// are some subtleties that we have not accounted for. people have worked
	// on this already. the document, "TIA Sounding off in the Digital Domain"
	// gives a good description of what's required.
	//
	// https://atariage.com/forums/topic/249865-tia-sounding-off-in-the-digital-domain/
	//
	// !TODO: simulate analogue sound generation
	return true, au.channel0.actualVol + au.channel1.actualVol
}
