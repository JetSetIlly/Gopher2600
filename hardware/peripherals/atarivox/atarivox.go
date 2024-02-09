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
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox/atarivoxengines"
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

type AtariVox struct {
	env *environment.Environment

	port plugging.PortID
	bus  ports.PeripheralBus

	swcha uint8

	// speakjet pins
	SpeakJetDATA  i2c.Trace
	SpeakJetREADY i2c.Trace

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
	logger.Logf("atarivox", "attached [%v]", vox.port)

	// attach savekey to same port
	vox.SaveKey = savekey.NewSaveKey(env, port, bus)

	return vox
}

func (vox *AtariVox) activateFestival() {
	if !vox.env.IsEmulation(environment.MainEmulation) {
		return
	}

	if vox.Engine != nil {
		vox.Engine.Quit()
		vox.Engine = nil
	}

	if vox.env.Prefs.AtariVox.FestivalEnabled.Get().(bool) {
		var err error

		vox.Engine, err = atarivoxengines.NewFestival(vox.env.Prefs.AtariVox.FestivalBinary.Get().(string))
		if err != nil {
			logger.Logf("atarivox", err.Error())
		}
	}
}

// Unplug implements the ports.Peripheral interface.
func (vox *AtariVox) Unplug() {
	vox.SaveKey.Unplug()
	if vox.Engine != nil {
		vox.Engine.Quit()
		vox.Engine = nil
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
					if vox.Engine != nil {
						vox.Engine.Flush()
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
			logger.Log("atarivox", "unexpected start bit of 1. should be 0")
			vox.State = AtariVoxStopped
		}
	case AtariVoxData:
		if vox.recvBit(vox.SpeakJetDATA.Hi()) {
			vox.State = AtariVoxEnding
		}
	case AtariVoxEnding:
		if vox.SpeakJetDATA.Hi() {
			vox.State = AtariVoxStopped
			if vox.Engine != nil && !(vox.disabled || vox.muted) {
				vox.Engine.SpeakJet(vox.Bits)
			}
		} else {
			logger.Log("atarivox", "unexpected end bit of 0. should be 1")
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
