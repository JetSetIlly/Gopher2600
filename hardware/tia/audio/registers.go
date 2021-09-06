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

import (
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// ReadMemRegisters checks the TIA memory for changes to registers that are
// interesting to the audio sub-system
//
// Returns true if memory.ChipData has not been serviced.
func (au *Audio) ReadMemRegisters(data bus.ChipData) bool {
	switch data.Name {
	case "AUDC0":
		au.channel0.regControl = data.Value & 0x0f
		au.channel0.reactAUDCx()
	case "AUDC1":
		au.channel1.regControl = data.Value & 0x0f
		au.channel1.reactAUDCx()
	case "AUDF0":
		au.channel0.regFreq = data.Value & 0x1f
		au.channel0.reactAUDCx()
	case "AUDF1":
		au.channel1.regFreq = data.Value & 0x1f
		au.channel1.reactAUDCx()
	case "AUDV0":
		au.channel0.regVolume = data.Value & 0x0f
		au.channel0.reactAUDCx()
	case "AUDV1":
		au.channel1.regVolume = data.Value & 0x0f
		au.channel1.reactAUDCx()
	default:
		return true
	}

	return false
}

// changing the value of an AUDx registers causes some side effect.
func (ch *channel) reactAUDCx() {
	freq := uint8(0)

	if ch.regControl == 0x00 || ch.regControl == 0x0b {
		ch.actualVol = ch.regVolume
	} else {
		freq = ch.regFreq

		// from TIASound.c: when bits D2 and D3 are set, the input source is
		// switched to the 1.19MHz clock, so the '30KHz' source clock is
		// reduced to approximately 10KHz."
		ch.useTenKhz = ch.regControl&0b1100 == 0b1100
	}

	if ch.freq != freq {
		// reset frequency if frequency has changed
		ch.freq = freq

		// if the channel is now "volume only" or was "volume only" ...
		if ch.freqCt == 0 || freq == 0 {
			// ... reset the counters
			ch.freqCt = 0
		}

		// ...otherwise let it complete the previous
	}
}
