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

package savekey

import (
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey/atarivoxengines"
	"github.com/jetsetilly/gopher2600/logger"
)

// AtariVoxState records how incoming signals to the AtariVox will be interpreted.
type AtariVoxState int

// List of valid AtariVoxStaate values.
const (
	AtariVoxStopped AtariVoxState = iota
	AtariVoxStarting
	AtariVoxData
	AtariVoxEnding
)

type AtariVox struct {
	// speakjet pins
	SpeakJetDATA  trace
	SpeakJetREADY trace

	State AtariVoxState

	// Data is sent by the VCS one bit at a time. see pushBits(), popBits() and
	// resetBits() for
	Bits   uint8
	BitsCt int

	// text to speech engine
	Engine atarivoxengines.AtariVoxEngine

	// slow down the rate at which stepAtariVox() operates once state is in the
	// Starting, Data or Ending state.
	baudCount int

	// once state has Stopped, the length of time to wait before flushing the
	// remaining bytes
	flushCount int
}

// recvBit will return true if bits field is full. the bits and bitsCt field
// will be reset on the next call.
func (vox *AtariVox) recvBit(v bool) bool {
	if vox.BitsCt >= 8 {
		vox.resetBits()
	}

	if v {
		// bits received from the other direction to the EEPROM
		vox.Bits |= 0x01 << vox.BitsCt
	}
	vox.BitsCt++

	return vox.BitsCt == 8
}

func (vox *AtariVox) resetBits() {
	vox.Bits = 0
	vox.BitsCt = 0
}

const (
	baudCount  = 62
	flushCount = 30000
)

func (sk *SaveKey) stepAtariVox() {
	if sk.AtariVox.Engine == nil {
		return
	}

	switch sk.AtariVox.State {
	case AtariVoxStopped:
		if sk.AtariVox.SpeakJetDATA.hi() {
			if sk.AtariVox.flushCount < flushCount {
				sk.AtariVox.flushCount++
				if sk.AtariVox.flushCount >= flushCount {
					sk.AtariVox.Engine.Flush()
				}
			}
			return
		}

		sk.AtariVox.resetBits()
		sk.AtariVox.State = AtariVoxStarting
		sk.AtariVox.baudCount = 0
		sk.AtariVox.flushCount = 0
	}

	sk.AtariVox.baudCount++
	if sk.AtariVox.baudCount < baudCount {
		return
	}
	sk.AtariVox.baudCount = 0

	switch sk.AtariVox.State {
	case AtariVoxStarting:
		if sk.AtariVox.SpeakJetDATA.lo() {
			sk.AtariVox.State = AtariVoxData
		} else {
			logger.Log("savekey", "unexpected start bit of 1. should be 0")
			sk.AtariVox.State = AtariVoxStopped
		}
	case AtariVoxData:
		if sk.AtariVox.recvBit(sk.AtariVox.SpeakJetDATA.hi()) {
			sk.AtariVox.State = AtariVoxEnding
		}
	case AtariVoxEnding:
		if sk.AtariVox.SpeakJetDATA.hi() {
			sk.AtariVox.State = AtariVoxStopped
			sk.AtariVox.Engine.SpeakJet(sk.AtariVox.Bits)
		} else {
			logger.Log("savekey", "unexpected end bit of 0. should be 1")
			sk.AtariVox.State = AtariVoxStopped
		}
	}
}
