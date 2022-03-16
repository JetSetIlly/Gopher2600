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

	"github.com/jetsetilly/gopher2600/hardware/instance"
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
	instance *instance.Instance

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
}

// NewAtariVox is the preferred method of initialisation for the AtariVox type.
func NewAtariVox(instance *instance.Instance, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	// there's no technical reason why the atarivox can't be attached to the
	// left player port but to keep things simple (we don't really want
	// multiple instances of an atarivox engine) we don't allow it
	//
	// moreover ROM developers understand that the atarivox is to be plugged
	// into the right player port and don't support left player port
	if port != plugging.PortRightPlayer {
		return nil
	}

	vox := &AtariVox{
		instance:      instance,
		port:          port,
		bus:           bus,
		SpeakJetDATA:  i2c.NewTrace(),
		SpeakJetREADY: i2c.NewTrace(),
	}

	var err error

	vox.Engine, err = atarivoxengines.NewFestival(vox.instance.Prefs.AtariVox.FestivalBinary.Get().(string))
	if err != nil {
		logger.Logf("atarivox", err.Error())
	}

	logger.Logf("atarivox", "attached [%v]", vox.port)

	// attach savekey to same port
	vox.SaveKey = savekey.NewSaveKey(instance, port, bus)

	return vox
}

// Periperhal is to be removed
func (vox *AtariVox) Unplug() {
	vox.SaveKey.Unplug()
	if vox.Engine != nil {
		vox.Engine.Quit()
	}
}

// Snapshot the instance of the Peripheral
func (vox *AtariVox) Snapshot() ports.Peripheral {
	n := *vox
	n.SaveKey = vox.SaveKey.Snapshot()
	return &n
}

// Plumb a new PeripheralBus into the Peripheral
func (vox *AtariVox) Plumb(bus ports.PeripheralBus) {
	vox.bus = bus
	vox.SaveKey.Plumb(bus)
}

// String should return information about the state of the peripheral
func (vox *AtariVox) String() string {
	return fmt.Sprintf("atarivox: %s", vox.SaveKey.String())
}

// The port the peripheral is plugged into
func (vox *AtariVox) PortID() plugging.PortID {
	return vox.port
}

// The ID of the peripheral being represented
func (vox *AtariVox) ID() plugging.PeripheralID {
	return plugging.PeriphAtariVox
}

// reset state of peripheral. this has nothing to do with the reset switch
// on the VCS panel
func (vox *AtariVox) Reset() {
	vox.SaveKey.Reset()
}

// the active bits in the SWCHA value.
const (
	maskSpeakJetDATA  = 0b00010000
	maskSpeakJetREADY = 0b00100000
)

const (
	baudCount  = 62
	flushCount = 30000
)

// memory has been updated. peripherals are notified.
func (vox *AtariVox) Update(data chipbus.ChangedRegister) bool {
	vox.SaveKey.Update(data)

	switch data.Register {
	case cpubus.SWCHA:
		// mask and shift SWCHA value to the normlised value
		switch vox.port {
		case plugging.PortLeftPlayer:
			vox.swcha = data.Value & 0xf0
		case plugging.PortRightPlayer:
			vox.swcha = (data.Value & 0x0f) << 4
		}
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

// step is called every CPU clock. important for paddle devices
func (vox *AtariVox) Step() {
	vox.SaveKey.Step()

	// update atarivox i2c state
	vox.SpeakJetDATA.Tick(vox.swcha&maskSpeakJetDATA == maskSpeakJetDATA)
	vox.SpeakJetREADY.Tick(vox.swcha&maskSpeakJetREADY == maskSpeakJetREADY)

	if vox.Engine == nil {
		return
	}

	switch vox.State {
	case AtariVoxStopped:
		if vox.SpeakJetDATA.Hi() {
			if vox.flushCount < flushCount {
				vox.flushCount++
				if vox.flushCount >= flushCount {
					vox.Engine.Flush()
				}
			}
			return
		}

		vox.resetBits()
		vox.State = AtariVoxStarting
		vox.baudCount = 0
		vox.flushCount = 0
	}

	// limit how often we update the atarivox - the successful 6507 program
	// will be written such that it fits in with this limitation
	vox.baudCount++
	if vox.baudCount < baudCount {
		return
	}
	vox.baudCount = 0

	switch vox.State {
	case AtariVoxStarting:
		if vox.SpeakJetDATA.Lo() {
			vox.State = AtariVoxData
		} else {
			logger.Log("savekey", "unexpected start bit of 1. should be 0")
			vox.State = AtariVoxStopped
		}
	case AtariVoxData:
		if vox.recvBit(vox.SpeakJetDATA.Hi()) {
			vox.State = AtariVoxEnding
		}
	case AtariVoxEnding:
		if vox.SpeakJetDATA.Hi() {
			vox.State = AtariVoxStopped
			vox.Engine.SpeakJet(vox.Bits)
		} else {
			logger.Log("savekey", "unexpected end bit of 0. should be 1")
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
	return false
}
