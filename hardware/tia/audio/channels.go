package audio

import (
	"fmt"
	"strings"
)

type channel struct {
	au *Audio

	regControl uint8
	regFreq    uint8
	regVolume  uint8

	poly4ct uint8
	poly5ct uint8
	poly9ct uint16

	divCt  uint8
	divMax uint8

	div3Ct uint8

	actualVol uint8
}

func (ch *channel) String() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%04b @ %05b ^ %04b", ch.regControl, ch.regFreq, ch.regVolume))
	return s.String()
}

func (ch *channel) process() {
	if ch.divCt > 1 {
		ch.divCt--
		return
	}

	if ch.divCt != 1 {
		return
	}

	var prevBit5 = ch.au.poly5bit[ch.poly5ct]

	ch.divCt = ch.divMax

	// from TIASound.c: "the P5 counter has multiple uses, so we inc it here"
	ch.poly5ct++
	if ch.poly5ct >= uint8(len(ch.au.poly5bit)) {
		ch.poly5ct = 0
	}

	// check for clock tick
	if (ch.regControl&0x02 == 0x0) ||
		((ch.regControl&0x01 == 0x0) && ch.au.div31[ch.poly5ct] != 0) ||
		((ch.regControl&0x01 == 0x1) && ch.au.poly5bit[ch.poly5ct] != 0) ||
		((ch.regControl&0x0f == 0xf) && ch.au.poly5bit[ch.poly5ct] == prevBit5) {

		if ch.regControl&0x04 == 0x04 {
			// use pure clock

			if ch.regControl&0x0f == 0x0f {
				// use poly5/div3
				if ch.au.poly5bit[ch.poly5ct] != prevBit5 {

					ch.div3Ct++
					if ch.div3Ct == 3 {
						ch.div3Ct = 0

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
				if ch.poly9ct >= uint16(len(ch.au.poly9bit)) {
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
			if ch.poly4ct >= uint8(len(ch.au.poly4bit)) {
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
