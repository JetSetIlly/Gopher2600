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

import "github.com/jetsetilly/gopher2600/hardware/memory/bus"

// UpdateRegisters checks the TIA memory for changes to registers that are
// interesting to the audio sub-system
//
// Returns true if memory.ChipData has not been serviced.
func (au *Audio) UpdateRegisters(data bus.ChipData) bool {
	switch data.Name {
	case "AUDC0":
		au.channel0.regControl = data.Value & 0x0f
	case "AUDC1":
		au.channel1.regControl = data.Value & 0x0f
	case "AUDF0":
		au.channel0.regFreq = data.Value & 0x1f
	case "AUDF1":
		au.channel1.regFreq = data.Value & 0x1f
	case "AUDV0":
		au.channel0.regVolume = data.Value & 0x0f
	case "AUDV1":
		au.channel1.regVolume = data.Value & 0x0f
	default:
		return true
	}

	au.channel0.reactAUDCx()
	au.channel1.reactAUDCx()

	return false
}

// changing the value of an AUDx registers causes some side effect.
func (ch *channel) reactAUDCx() {
	freqClk := uint8(0)

	if ch.regControl == 0x00 || ch.regControl == 0x0b {
		ch.actualVol = ch.regVolume
	} else {
		freqClk = ch.regFreq + 1

		// from TIASound.c: when bits D2 and D3 are set, the input source is
		// switched to the 1.19MHz clock, so the '30KHz' source clock is
		// reduced to approximately 10KHz."
		//
		// multiplying the frequency clock by 3 effectively divides clock114 by
		// a further 3, giving us the 10Khz clock.
		if ch.regControl&0x0c == 0x0c && ch.regControl != 0x0f {
			freqClk *= 3
		}
	}

	if freqClk != ch.adjFreq {
		// reset frequency if frequency has changed
		ch.adjFreq = freqClk

		// if the channel is now "volume only" or was "volume only" ...
		if ch.freqClk == 0 || freqClk == 0 {
			// ... reset the counter
			ch.freqClk = freqClk
		}

		// ...otherwide let it complete the previous
	}
}
