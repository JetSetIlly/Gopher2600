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
	"strings"
	"unicode"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey/i2c"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/logger"
)

// SaveKeyState records how incoming signals to the SaveKey will be interpreted.
type SaveKeyState int

// List of valid SaveKeyState values.
const (
	SaveKeyStopped SaveKeyState = iota
	SaveKeyStarting
	SaveKeyAddressHi
	SaveKeyAddressLo
	SaveKeyData
)

// DataDirection indicates the direction of data flow between the VCS and the SaveKey.
type DataDirection int

// Valid DataDirection values.
const (
	Reading DataDirection = iota
	Writing
)

// SaveKey represents the SaveKey peripheral. It implements the Peripheral
// interface.
type SaveKey struct {
	port plugging.PortID
	bus  ports.PeripheralBus

	// the i2c protocol used by the SaveKey is transferred via the SWCHA
	// register from the CPU. we keep a copy of the SWCHA value set in the
	// Update() function, so we can reuse it in the Step() function
	swcha uint8

	// only two bits of the SWCHA value is of interest to the i2c protocol.
	// from the perspective of the second player (in which port the SaveKey is
	// usually inserted) pin 2 is the data signal (SDA) and pin 3 is the
	// clock signal (SCL)
	SDA i2c.Trace
	SCL i2c.Trace

	// incoming data is interpreted depending on the state of the i2c protocol.
	// we also need to know the direction of data flow at any given time and
	// whether the next bit should be acknowledged (the Ack bool)
	State SaveKeyState
	Dir   DataDirection
	Ack   bool

	// Data is sent by the VCS one bit at a time. see pushBits(), popBits() and
	// resetBits() for
	Bits   uint8
	BitsCt int

	// the core of the SaveKey is an EEPROM.
	EEPROM *EEPROM
}

// NewSaveKey is the preferred method of initialisation for the SaveKey type.
func NewSaveKey(env *environment.Environment, port plugging.PortID, bus ports.PeripheralBus) ports.Peripheral {
	// there's no technical reason why the savekey can't be attached to the
	// left player port but to keep things simple (write contention on the
	// savekey file) we don't allow it
	//
	// moreover ROM developers understand that the savekey is to be
	// plugged into the right player port and don't support left player port
	if port != plugging.PortRight {
		return nil
	}

	sk := &SaveKey{
		port:   port,
		bus:    bus,
		SDA:    i2c.NewTrace(),
		SCL:    i2c.NewTrace(),
		State:  SaveKeyStopped,
		EEPROM: newEeprom(),
	}

	sk.bus.WriteSWCHx(sk.port, 0xf0)
	logger.Logf("savekey", "attached [%v]", sk.port)

	return sk
}

// Unplug implements the Peripheral interface.
func (sk *SaveKey) Unplug() {
}

// Snapshot implements the Peripheral interface.
func (sk *SaveKey) Snapshot() ports.Peripheral {
	n := *sk
	return &n
}

// Plumb implements the ports.Peripheral interface.
func (sk *SaveKey) Plumb(bus ports.PeripheralBus) {
	sk.bus = bus
}

func (sk *SaveKey) String() string {
	s := strings.Builder{}
	s.WriteString("savekey: ")

	switch sk.State {
	case SaveKeyStopped:
		s.WriteString("stopped")
	case SaveKeyStarting:
		s.WriteString("starting")
	case SaveKeyAddressHi:
		fallthrough
	case SaveKeyAddressLo:
		s.WriteString("address")
	case SaveKeyData:
		switch sk.Dir {
		case Reading:
			s.WriteString("reading ")
		case Writing:
			s.WriteString("writing ")
		}
		s.WriteString("data")
	}

	if sk.Ack {
		s.WriteString(" [ACK]")
	}

	return s.String()
}

// PortID implements the ports.Peripheral interface.
func (sk *SaveKey) PortID() plugging.PortID {
	return sk.port
}

// ID implements the ports.Peripheral interface.
func (sk *SaveKey) ID() plugging.PeripheralID {
	return plugging.PeriphSavekey
}

// Reset implements the ports.Peripheral interface.
func (sk *SaveKey) Reset() {
}

// the active bits in the SWCHA value.
const (
	maskSaveKeySDA = 0b01000000
	maskSaveKeySCL = 0b10000000
)

// the bit sequence to indicate read/write data direction.
const (
	writeSig = 0xa0
	readSig  = 0xa1
)

// Update implements the ports.Peripheral interface.
func (sk *SaveKey) Update(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.SWCHA:
		// mask and shift SWCHA value to the normlised value
		switch sk.port {
		case plugging.PortLeft:
			sk.swcha = data.Value & 0xf0
		case plugging.PortRight:
			sk.swcha = (data.Value & 0x0f) << 4
		}

	default:
		return true
	}

	return false
}

// recvBit will return true if bits field is full. the bits and bitsCt field
// will be reset on the next call.
func (sk *SaveKey) recvBit(v bool) bool {
	if sk.BitsCt >= 8 {
		sk.resetBits()
	}

	if v {
		sk.Bits |= 0x01 << (7 - sk.BitsCt)
	}
	sk.BitsCt++

	return sk.BitsCt == 8
}

// return the next bit in the current byte. end is true if all bits in the
// current byte has been exhausted. next call to sendBit() will use the next
// byte in the EEPROM page.
func (sk *SaveKey) sendBit() (bit bool, end bool) {
	if sk.BitsCt >= 8 {
		sk.resetBits()
	}

	if sk.BitsCt == 0 {
		sk.Bits = sk.EEPROM.get()
	}

	v := (sk.Bits >> (7 - sk.BitsCt)) & 0x01
	bit = v == 0x01
	sk.BitsCt++

	if sk.BitsCt >= 8 {
		end = true
	}

	return bit, end
}

func (sk *SaveKey) resetBits() {
	sk.Bits = 0
	sk.BitsCt = 0
}

// Step implements the ports.Peripheral interface.
func (sk *SaveKey) Step() {
	// update savekey i2c state
	sk.SDA.Tick(sk.swcha&maskSaveKeySDA == maskSaveKeySDA)
	sk.SCL.Tick(sk.swcha&maskSaveKeySCL == maskSaveKeySCL)

	// check for stop signal before anything else
	if sk.State != SaveKeyStopped && sk.SCL.Hi() && sk.SDA.Rising() {
		logger.Log("savekey", "stopped message")
		sk.State = SaveKeyStopped
		sk.EEPROM.Write()
		return
	}

	// if SCL is not changing to a hi state then we don't need to do anything
	if !sk.SCL.Rising() {
		return
	}

	// if the VCS is waiting for an ACK then handle that now
	if sk.Ack {
		if sk.Dir == Reading && sk.SDA.Falling() {
			sk.bus.WriteSWCHx(sk.port, maskSaveKeySDA)
			sk.Ack = false
			return
		}
		sk.bus.WriteSWCHx(sk.port, 0x00)
		sk.Ack = false
		return
	}

	// interpret i2c state depending on which state we are currently in
	switch sk.State {
	case SaveKeyStopped:
		if sk.SDA.Lo() {
			logger.Log("savekey", "starting message")
			sk.resetBits()
			sk.State = SaveKeyStarting
		}

	case SaveKeyStarting:
		if sk.recvBit(sk.SDA.Falling()) {
			switch sk.Bits {
			case readSig:
				logger.Log("savekey", "reading message")
				sk.resetBits()
				sk.State = SaveKeyData
				sk.Dir = Reading
				sk.Ack = true
			case writeSig:
				logger.Log("savekey", "writing message")
				sk.State = SaveKeyAddressHi
				sk.Dir = Writing
				sk.Ack = true
			default:
				logger.Log("savekey", "unrecognised message")
				sk.State = SaveKeyStopped
			}
		}

	case SaveKeyAddressHi:
		if sk.recvBit(sk.SDA.Falling()) {
			sk.EEPROM.Address = uint16(sk.Bits) << 8
			sk.State = SaveKeyAddressLo
			sk.Ack = true
		}

	case SaveKeyAddressLo:
		if sk.recvBit(sk.SDA.Falling()) {
			sk.EEPROM.Address |= uint16(sk.Bits)
			sk.State = SaveKeyData
			sk.Ack = true

			switch sk.Dir {
			case Reading:
				logger.Logf("savekey", "reading from address %#04x", sk.EEPROM.Address)
			case Writing:
				logger.Logf("savekey", "writing to address %#04x", sk.EEPROM.Address)
			}
		}

	case SaveKeyData:
		switch sk.Dir {
		case Reading:
			bit, end := sk.sendBit()

			if bit {
				sk.bus.WriteSWCHx(sk.port, maskSaveKeySDA)
			} else {
				sk.bus.WriteSWCHx(sk.port, 0x00)
			}

			if end {
				if unicode.IsPrint(rune(sk.Bits)) {
					logger.Logf("savekey", "read byte %#02x [%c]", sk.Bits, sk.Bits)
				} else {
					logger.Logf("savekey", "read byte %#02x", sk.Bits)
				}
				sk.Ack = true
			}

		case Writing:
			if sk.recvBit(sk.SDA.Falling()) {
				if unicode.IsPrint(rune(sk.Bits)) {
					logger.Logf("savekey", "written byte %#02x [%c]", sk.Bits, sk.Bits)
				} else {
					logger.Logf("savekey", "written byte %#02x", sk.Bits)
				}
				sk.EEPROM.put(sk.Bits)
				sk.Ack = true
			}
		}
	}
}

// HandleEvent implements the ports.Peripheral interface.
func (sk *SaveKey) HandleEvent(_ ports.Event, _ ports.EventData) (bool, error) {
	return false, nil
}

// IsActive implements the ports.Peripheral interface.
func (sk *SaveKey) IsActive() bool {
	return false
}
