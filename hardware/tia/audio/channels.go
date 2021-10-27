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

	// which bit of each polynomial counter to use next
	poly4ct int
	poly5ct int
	poly9ct int
	div3ct  uint8

	// the different musical notes available to the 2600 are achieved with a
	// frequency clock. the easiest way to think of this is to think of a
	// filter to the 30Khz clock signal.
	freqCt uint8

	// if bits 2 and 3 of control register are set (ie. mask 0x0c) then we use
	// a 10Khz clock rather than a 30Khz clock.
	useTenKhz bool

	// the frequency of the channel. this is either the value that's in the
	// frequency register (regFreq) or zero if the volume bits are set in the
	// control register (regControl).
	//
	// this is the value we count to with freqCt in order to generate the
	// correct sound
	freq uint8

	// the different tones are achieved are by adjusting the volume between
	// zero (silence) and the value in the volume register. actualVol is a
	// record of that value.
	actualVol uint8
}

func (ch *channel) String() string {
	return ch.registers.String()
}

// tick should be called at a frequency of 30Khz. when the 10Khz clock is
// required, the frequency clock is increased by a factor of three.
func (ch *channel) tick(tenKhz bool) {
	// filter out 30Khz signal if channel is set to use the 10Khz signal
	if ch.useTenKhz && !tenKhz {
		return
	}

	// nothing to do except change the volume if control register is 0x00.
	// volume has already been changed with reactAUDCx() which is called
	// whenever an audio register is changed
	if ch.registers.Control == 0x00 {
		return
	}

	// tick main frequency clock
	if ch.freqCt == ch.freq || ch.freqCt == 31 {
		ch.freqCt = 0
	} else {
		ch.freqCt++
	}

	// update output volume only when the counter reaches the target frequency value
	if ch.freqCt != ch.freq {
		return
	}

	// the 5-bit polynomial clock toggles volume on change of bit. note the
	// current bit so we can compare
	var prevBit5 = poly5bit[ch.poly5ct]

	// advance 5-bit polynomial clock
	ch.poly5ct++
	if ch.poly5ct >= len(poly5bit) {
		ch.poly5ct = 0
	}

	// check for clock tick
	if (ch.registers.Control&0x02 == 0x0) ||
		((ch.registers.Control&0x01 == 0x0) && div31[ch.poly5ct] != 0) ||
		((ch.registers.Control&0x01 == 0x1) && poly5bit[ch.poly5ct] != 0) ||
		((ch.registers.Control&0x0f == 0xf) && poly5bit[ch.poly5ct] != prevBit5) {

		if ch.registers.Control&0x04 == 0x04 {
			// use pure clock

			if ch.registers.Control&0x0f == 0x0f {
				// use poly5/div3
				if poly5bit[ch.poly5ct] != prevBit5 {
					ch.div3ct++
					if ch.div3ct == 3 {
						ch.div3ct = 0

						// toggle volume
						if ch.actualVol != 0 {
							ch.actualVol = 0
						} else {
							ch.actualVol = ch.registers.Volume
						}
					}
				}
			} else {
				// toggle volume
				if ch.actualVol != 0 {
					ch.actualVol = 0
				} else {
					ch.actualVol = ch.registers.Volume
				}
			}
		} else if ch.registers.Control&0x08 == 0x08 {
			// use poly poly5/poly9

			if ch.registers.Control == 0x08 {
				// use poly9
				ch.poly9ct++
				if ch.poly9ct >= len(poly9bit) {
					ch.poly9ct = 0
				}

				// toggle volume
				if poly9bit[ch.poly9ct] != 0 {
					ch.actualVol = ch.registers.Volume
				} else {
					ch.actualVol = 0
				}
			} else if ch.registers.Control&0x02 != 0 {
				if ch.actualVol != 0 || ch.registers.Control&0x01 == 0x01 {
					ch.actualVol = 0
				} else {
					ch.actualVol = ch.registers.Volume
				}
			} else {
				// use poly5. we've already bumped poly5 counter forward

				// toggle volume
				if poly5bit[ch.poly5ct] == 1 {
					ch.actualVol = ch.registers.Volume
				} else {
					ch.actualVol = 0
				}
			}
		} else {
			// use poly 4
			ch.poly4ct++
			if ch.poly4ct >= len(poly4bit) {
				ch.poly4ct = 0
			}

			if poly4bit[ch.poly4ct] == 1 {
				ch.actualVol = ch.registers.Volume
			} else {
				ch.actualVol = 0
			}
		}
	}
}
