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

type channel struct {
	registers        Registers
	registersChanged bool

	clockEnable      bool
	noiseFeedback    bool
	noiseCounterBit4 bool
	pulseCounterHold bool

	divCounter   uint8
	pulseCounter uint8
	noiseCounter uint8

	actualVol uint8
}

func (ch *channel) String() string {
	return ch.registers.String()
}

// tick should be called at a frequency of 30Khz. when the 10Khz clock is
// required, the frequency clock is increased by a factor of three.
func (ch *channel) tick() {
	ch.phase0()
	ch.phase1()
}

func (ch *channel) phase0() {
	// phase 0

	if ch.clockEnable {
		ch.noiseCounterBit4 = ch.noiseCounter&0x01 != 0x00

		switch ch.registers.Control & 0x03 {
		case 0x00:
			fallthrough
		case 0x01:
			ch.pulseCounterHold = false
		case 0x02:
			ch.pulseCounterHold = ch.noiseCounter&0x1e != 0x02
		case 0x03:
			ch.pulseCounterHold = !ch.noiseCounterBit4
		}

		switch ch.registers.Control & 0x03 {
		case 0x00:
			ch.noiseFeedback = (((ch.pulseCounter ^ ch.noiseCounter) & 0x01) != 0x00) ||
				!(ch.noiseCounter != 0x00 || ch.pulseCounter != 0x0a) ||
				(ch.registers.Control&0x0c == 0x00)
		default:
			var n uint8
			if ch.noiseCounter&0x04 != 0x00 {
				n = 1
			}
			ch.noiseFeedback = (n^(ch.noiseCounter&0x01) != 0x00) || ch.noiseCounter == 0
		}
	}

	ch.clockEnable = ch.divCounter == ch.registers.Freq

	if ch.divCounter == ch.registers.Freq || ch.divCounter == 0x1f {
		ch.divCounter = 0
	} else {
		ch.divCounter++
	}
}

func (ch *channel) phase1() {
	// phase 1

	if ch.clockEnable {
		pulseFeedback := false

		switch ch.registers.Control >> 2 {
		case 0x00:
			var n uint8
			if ch.pulseCounter&0x02 != 0x00 {
				n = 1
			}
			pulseFeedback = (n^(ch.pulseCounter&0x01) != 0x00) &&
				(ch.pulseCounter != 0x0a) &&
				(ch.registers.Control&0x03 != 0x00)
		case 0x01:
			pulseFeedback = ch.pulseCounter&0x08 == 0x00
		case 0x02:
			pulseFeedback = !ch.noiseCounterBit4
		case 0x03:
			pulseFeedback = !((ch.pulseCounter&0x02 != 0x00) || (ch.pulseCounter&0x0e == 0x00))
		}

		ch.noiseCounter >>= 1

		if ch.noiseFeedback {
			ch.noiseCounter |= 0x10
		}

		if !ch.pulseCounterHold {
			ch.pulseCounter = ^(ch.pulseCounter >> 1) & 0x07

			if pulseFeedback {
				ch.pulseCounter |= 0x08
			}
		}
	}

	ch.actualVol = (ch.pulseCounter & 0x01) * ch.registers.Volume
}
