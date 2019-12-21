package audio

import (
	"fmt"
	"strings"
)

type channel struct {
	au *Audio

	// each channel has three registers that control its output. from the
	// "Stella Programmer's Guide":
	//
	// "Each audio circuit has three registers that control a noise-tone
	// generator (what kind of sound), a frequency selection (high or low pitch
	// of the sound), and a volume control."
	//
	// not all the bits are used in each register. the comments below indicate
	// how many of the least-significant bits are used.
	regControl uint8 // 4 bit
	regFreq    uint8 // 5 bit
	regVolume  uint8 // 4 bit

	// which bit of each polynomial counter to use next
	poly4ct int
	poly5ct int
	poly9ct int

	// the different musical notes available to the 2600 are achived with a
	// frequency clock. the easiest way to think of this is to think of a
	// filter to the 30Khz clock signal.
	freqClk uint8

	div3ct uint8

	// the adjusted frequency is the value of the frequency register. when the
	// 10KHz clock is required, this value is increased by a factor of 3
	adjFreq uint8

	// the different tones are achieved are by adjusting the volume between
	// zero (silence) and the value in the volume register. actualVol is a
	// record of that value.
	actualVol uint8
}

func (ch *channel) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%04b @ %05b ^ %04b", ch.regControl, ch.regFreq, ch.regVolume))
	return s.String()
}

// tick should be called at a frequency of 30Khz. when the 10Khz clock is
// required, the frequency clock is increased by a factor of three.
func (ch *channel) tick() {
	// tick frequency clock
	if ch.freqClk > 1 {
		ch.freqClk--
		return
	}

	if ch.freqClk != 1 {
		return
	}

	// when frequency clock reaches zero, reset it back to the adjusted
	// frequency value
	ch.freqClk = ch.adjFreq

	// the 5-bit polynomial clock toggles volume on change of bit. note the
	// current bit so we can compare
	var prevBit5 = ch.au.poly5bit[ch.poly5ct]

	// advance 5-bit polynomial clock
	ch.poly5ct++
	if ch.poly5ct >= len(ch.au.poly5bit) {
		ch.poly5ct = 0
	}

	// check for clock tick
	if (ch.regControl&0x02 == 0x0) ||
		((ch.regControl&0x01 == 0x0) && ch.au.div31[ch.poly5ct] != 0) ||
		((ch.regControl&0x01 == 0x1) && ch.au.poly5bit[ch.poly5ct] != 0) ||
		((ch.regControl&0x0f == 0xf) && ch.au.poly5bit[ch.poly5ct] != prevBit5) {

		if ch.regControl&0x04 == 0x04 {
			// use pure clock

			if ch.regControl&0x0f == 0x0f {
				// use poly5/div3
				if ch.au.poly5bit[ch.poly5ct] != prevBit5 {
					ch.div3ct++
					if ch.div3ct == 3 {
						ch.div3ct = 0

						// toggle volume
						if ch.actualVol != 0 {
							ch.actualVol = 0
						} else {
							ch.actualVol = ch.regVolume
						}
					}
				}
			} else {
				// toggle volume
				if ch.actualVol != 0 {
					ch.actualVol = 0
				} else {
					ch.actualVol = ch.regVolume
				}
			}

		} else if ch.regControl&0x08 == 0x08 {
			// use poly poly5/poly9

			if ch.regControl == 0x08 {
				// use poly9
				ch.poly9ct++
				if ch.poly9ct >= len(ch.au.poly9bit) {
					ch.poly9ct = 0
				}

				// toggle volume
				if ch.au.poly9bit[ch.poly9ct] != 0 {
					ch.actualVol = ch.regVolume
				} else {
					ch.actualVol = 0
				}
			} else if ch.regControl&0x02 != 0 {
				if ch.actualVol != 0 || ch.regControl&0x01 == 0x01 {
					ch.actualVol = 0
				} else {
					ch.actualVol = ch.regVolume
				}
			} else {
				// use poly5. we've already bumped poly5 counter forward

				// toggle volume
				if ch.au.poly5bit[ch.poly5ct] == 1 {
					ch.actualVol = ch.regVolume
				} else {
					ch.actualVol = 0
				}
			}
		} else {
			// use poly 4
			ch.poly4ct++
			if ch.poly4ct >= len(ch.au.poly4bit) {
				ch.poly4ct = 0
			}

			if ch.au.poly4bit[ch.poly4ct] == 1 {
				ch.actualVol = ch.regVolume
			} else {
				ch.actualVol = 0
			}
		}
	}
}
