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

package atarivox

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox/engines"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox/msa"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey/i2c"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
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

// the maximum number of bytes in the SpeakJetStream slice
const maxSpeakJetStream = 128

type AtariVox struct {
	env *environment.Environment

	port plugging.PortID
	bus  ports.PeripheralBus

	swcha uint8

	// current state of the atarivox peripheal
	State AtariVoxState

	// speakjet pins
	SpeakJetDATA  i2c.Trace
	SpeakJetREADY i2c.Trace

	// stream of date interpreted as speakjet commands
	SpeakJetStream []uint8

	// data is sent by the VCS one bit at a time. see pushBits(), popBits() and
	// resetBits() for
	bits   uint8
	bitsCt int

	pushed     bool
	pushedBits uint8

	// text to speech engine
	engine engines.AtariVoxEngine

	// slow down the rate at which stepAtariVox() operates once state is in the
	// Starting, Data or Ending state.
	baudCount int

	// once state has Stopped, the length of time to wait before flushing the
	// remaining bytes
	flushCount int

	// the savekey portion of the AtariVox is the same as a stand alone savekey
	SaveKey ports.Peripheral

	// the atarivox should not process data when it is disabled (see comment
	// for mute below)
	disabled bool

	// the atarivox should not process data when it is muted. slightly
	// different flag to disabled because they are set via different code
	// paths. muted is intended to follow the wider audio mute settings. the
	// disabled flag meanwhile is independent of muting and allows AtariVox to
	// be turned off even when emulation wide audio is not muted.
	muted bool
}

// NewAtariVox is the preferred method of initialisation for the AtariVox type.
func NewAtariVox(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	// there's no technical reason why the atarivox can't be attached to the
	// left player port but to keep things simple (we don't really want
	// multiple instances of an atarivox engine) we don't allow it
	//
	// moreover ROM developers understand that the atarivox is to be plugged
	// into the right player port and don't support left player port
	if port != plugging.PortRight {
		return nil
	}

	vox := &AtariVox{
		env:           env,
		port:          port,
		bus:           bus,
		SpeakJetDATA:  i2c.NewTrace(),
		SpeakJetREADY: i2c.NewTrace(),
	}

	vox.activateFestival()
	logger.Logf(env, "atarivox", "attached [%v]", vox.port)

	// attach savekey to same port
	vox.SaveKey = savekey.NewSaveKey(env, port, bus)

	return vox
}

func (vox *AtariVox) activateFestival() {
	if !vox.env.IsEmulation(environment.MainEmulation) {
		return
	}

	if vox.engine != nil {
		vox.engine.Quit()
		vox.engine = nil
	}

	if vox.env.Prefs.AtariVox.FestivalEnabled.Get().(bool) {
		var err error

		vox.engine, err = engines.NewFestival(vox.env)
		if err != nil {
			logger.Log(vox.env, "atarivox", err)
		}
	}
}

// Unplug implements the ports.Peripheral interface.
func (vox *AtariVox) Unplug() {
	vox.SaveKey.Unplug()
	if vox.engine != nil {
		vox.engine.Quit()
		vox.engine = nil
	}
}

// Snapshot implements the ports.Peripheral interface.
func (vox *AtariVox) Snapshot() ports.Peripheral {
	n := *vox
	n.SaveKey = vox.SaveKey.Snapshot()
	return &n
}

// Plumb implements the ports.Peripheral interface.
func (vox *AtariVox) Plumb(bus ports.PeripheralBus) {
	vox.bus = bus
	vox.SaveKey.Plumb(bus)
}

// String implements the ports.Peripheral interface.
func (vox *AtariVox) String() string {
	return fmt.Sprintf("atarivox: %s", vox.SaveKey.String())
}

// PortID implements the ports.Peripheral interface.
func (vox *AtariVox) PortID() plugging.PortID {
	return vox.port
}

// ID implements the ports.Peripheral interface.
func (vox *AtariVox) ID() plugging.PeripheralID {
	return plugging.PeriphAtariVox
}

// Reset implements the ports.Peripheral interface.
func (vox *AtariVox) Reset() {
	// nothing to do for the atarivox but we forward the reset signal to the savekey
	vox.SaveKey.Reset()
}

// Restart implements the ports.RestartPeripheral interface.
func (vox *AtariVox) Restart() {
	vox.activateFestival()
}

// Restart implements the ports.DisablePeripheral interface.
func (vox *AtariVox) Disable(disabled bool) {
	vox.disabled = disabled
}

// Mute silences atarivox output for the duration muted is true.
//
// This implements a private mutePeripheral interface in the ports package. It
// should not be called directly except via the Mute() function in the Ports
// implementation.
func (vox *AtariVox) Mute(muted bool) {
	vox.muted = muted
}

// the active bits in the SWCHA value.
const (
	maskSpeakJetDATA  = 0b00010000
	maskSpeakJetREADY = 0b00100000
)

const (
	baudCount  = 62
	flushCount = 5000
)

// memory has been updated. peripherals are notified.
func (vox *AtariVox) Update(data chipbus.ChangedRegister) bool {
	vox.SaveKey.Update(data)

	switch data.Register {
	case cpubus.SWCHA:
		// mask and shift SWCHA value to the normlised value
		switch vox.port {
		case plugging.PortLeft:
			vox.swcha = data.Value & 0xf0
		case plugging.PortRight:
			vox.swcha = (data.Value & 0x0f) << 4
		}

	default:
		return true
	}

	return false
}

// recvBit will return true if bits field is full. the bits and bitsCt field
// will be reset on the next call.
func (vox *AtariVox) recvBit(v bool) bool {
	if vox.bitsCt >= 8 {
		vox.resetBits()
	}

	if v {
		// bits received from the other direction to the EEPROM
		vox.bits |= 0x01 << vox.bitsCt
	}
	vox.bitsCt++

	return vox.bitsCt == 8
}

func (vox *AtariVox) resetBits() {
	vox.bits = 0
	vox.bitsCt = 0
}

// Step is called every CPU clock.
func (vox *AtariVox) Step() {
	vox.SaveKey.Step()

	if vox.SaveKey.IsActive() {
		return
	}

	// limit how often we update the atarivox - the successful 6507 program
	// will be written such that it fits in with this limitation
	vox.baudCount++
	if vox.baudCount < baudCount {
		return
	}
	vox.baudCount = 0

	// update atarivox i2c state
	vox.SpeakJetDATA.Tick(vox.swcha&maskSpeakJetDATA == maskSpeakJetDATA)
	vox.SpeakJetREADY.Tick(vox.swcha&maskSpeakJetREADY == maskSpeakJetREADY)

	switch vox.State {
	case AtariVoxStopped:
		if vox.SpeakJetDATA.Hi() {
			if vox.flushCount < flushCount {
				vox.flushCount++
				if vox.flushCount >= flushCount {
					if vox.engine != nil {
						vox.engine.Flush()
					}
				}
			}
			return
		}

		vox.resetBits()
		vox.State = AtariVoxStarting
		vox.baudCount = 0
		vox.flushCount = 0
	}

	switch vox.State {
	case AtariVoxStarting:
		if vox.SpeakJetDATA.Lo() {
			vox.State = AtariVoxData
		} else {
			logger.Log(vox.env, "atarivox", "unexpected start bit of 1. should be 0")
			vox.State = AtariVoxStopped
		}
	case AtariVoxData:
		if vox.recvBit(vox.SpeakJetDATA.Hi()) {
			vox.State = AtariVoxEnding
		}
	case AtariVoxEnding:
		if vox.SpeakJetDATA.Hi() {
			vox.State = AtariVoxStopped

			// some speakjet commands require 16 bits. for these commands we pushe the current 8
			// bits and send them after the next 8 bits, interpretting those next 8 bits as data
			var pushed bool

			// the following condition could be expressed more simply but a large switch like this
			// allows for more commentary and also provides a structure in case additional logic is
			// ever required
			//
			// note that we don't set the pushed flag if the previous command was pushed. this is so
			// that we don't interpret data incorrectly - ie. the data payload for any of these
			// commands can be any value
			switch vox.bits {
			case 20: // volume
				pushed = !vox.pushed
			case 21: // speed
				pushed = !vox.pushed
			case 22: // pitch
				pushed = !vox.pushed
			case 23: // bend
				pushed = !vox.pushed
			case 24: // PortCtr
				pushed = !vox.pushed
			case 25: // Port
				pushed = !vox.pushed
			case 26: // Repeat
				pushed = !vox.pushed
			case 28: // Call Phrase
				pushed = !vox.pushed
			case 29: // Goto Phrase
				pushed = !vox.pushed
			case 30: // Delay
				pushed = !vox.pushed
			}

			if pushed {
				vox.pushed = true
				vox.pushedBits = vox.bits
			} else {
				if vox.engine != nil && !(vox.disabled || vox.muted) {
					if vox.pushed {
						vox.engine.SpeakJet(vox.pushedBits, vox.bits)
					} else {
						vox.engine.SpeakJet(vox.bits, 0)
					}
					vox.SpeakJetStream = append(vox.SpeakJetStream, vox.bits)

					// log atarivox phoneme
					switch c := msa.Commands[vox.bits].(type) {
					case msa.Allophone:
						logger.Log(vox.env, "atarivox", c.Phoneme)
					}

					if len(vox.SpeakJetStream) > maxSpeakJetStream {
						// we've only added one byte to the stream so we probably only need to crop one byte
						vox.SpeakJetStream = vox.SpeakJetStream[1:]

						// however, that one byte might be a double-byte command, in which case we need
						// to crop two bytes
						switch c := msa.Commands[vox.SpeakJetStream[0]].(type) {
						case msa.ControlCode:
							if c.Double {
								vox.SpeakJetStream = vox.SpeakJetStream[1:]
							}
						}
					}
				}

				// always reset pushed flag, even if engine is disabled, muted or missing
				vox.pushed = false
			}
		} else {
			logger.Log(vox.env, "atarivox", "unexpected end bit of 0. should be 1")
			vox.State = AtariVoxStopped
		}
	}
}

// handle an incoming input event
func (vox *AtariVox) HandleEvent(_ ports.Event, _ ports.EventData) (bool, error) {
	return false, nil
}

// whether the peripheral is currently "active"
func (vox *AtariVox) IsActive() bool {
	return vox.State != AtariVoxStopped
}
