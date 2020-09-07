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
	"fmt"
	"os"
	"unicode"

	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/paths"
)

const saveKeyPath = "savekey"

type messageState int

const (
	stopped messageState = iota
	starting
	addressHi
	addressLo
	data
)

type ackState int

const (
	none ackState = iota
	ack
)

type directionState int

const (
	reading directionState = iota
	writing
)

type SaveKey struct {
	id  ports.PortID
	bus ports.PeripheralBus

	signal uint8

	SDA trace
	SCL trace

	state messageState
	ack   ackState
	dir   directionState

	bits   uint8
	bitsCt int

	address uint16

	data []uint8
}

func NewSaveKey(id ports.PortID, bus ports.PeripheralBus) ports.Peripheral {
	sk := &SaveKey{
		id:    id,
		bus:   bus,
		SDA:   newTrace(),
		SCL:   newTrace(),
		state: stopped,
		data:  make([]uint8, 0x10000),
	}

	// initialise data with 0xff
	for i := range sk.data {
		sk.data[i] = 0xff
	}

	sk.bus.WriteSWCHx(sk.id, 0xf0)
	logger.Log("savekey", fmt.Sprintf("savekey attached [%s]", sk.id.String()))

	sk.loadSaveKey()

	return sk
}

func (sk *SaveKey) loadSaveKey() {
	fn, err := paths.ResourcePath("", saveKeyPath)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not load savekey file (%s)", err))
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not load savekey file (%s)", err))
		return
	}
	defer f.Close()

	// get file info. not using Stat() on the file handle because the
	// windows version (when running under wine) does not handle that
	fs, err := os.Stat(fn)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not load savekey file (%s)", err))
		return
	}
	if fs.Size() != int64(len(sk.data)) {
		logger.Log("savekey", fmt.Sprintf("savekey file is of incorrect length. %d should be 65536 ", fs.Size()))
	}

	_, err = f.Read(sk.data)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not load savekey file (%s)", err))
		return
	}

	logger.Log("savekey", fmt.Sprintf("savekey file loaded from %s", fn))
}

func (sk *SaveKey) writeSaveKey() {
	fn, err := paths.ResourcePath("", saveKeyPath)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not write savekey file (%s)", err))
		return
	}

	f, err := os.Create(fn)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not write savekey file (%s)", err))
		return
	}
	defer f.Close()

	n, err := f.Write(sk.data)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not write savekey file (%s)", err))
		return
	}

	if n != len(sk.data) {
		logger.Log("savekey", fmt.Sprintf("savekey file has not been truncated during write. %d should be 65536", n))
		return
	}

	logger.Log("savekey", fmt.Sprintf("savekey file saved to %s", fn))
}

func (sk *SaveKey) String() string {
	return "TODO: savekey info"
}

func (sk *SaveKey) Name() string {
	return "SaveKey"
}

func (sk *SaveKey) Reset() {
}

const maskSDA = 0b01000000
const maskSCL = 0b10000000

const writeSig = 0xa0
const readSig = 0xa1

func (sk *SaveKey) Update(data bus.ChipData) bool {
	switch data.Name {
	case "SWCHA":
		// mask and shift SWCHA value to the normlised value
		switch sk.id {
		case ports.Player0ID:
			sk.signal = data.Value & 0xf0
		case ports.Player1ID:
			sk.signal = (data.Value & 0x0f) << 4
		}
	}

	return false
}

// pushBit will return true if bits field is full. the bits and bitsCt field
// will be reset on the next call.
func (sk *SaveKey) pushBit(v bool) bool {
	if sk.bitsCt >= 8 {
		sk.resetBits()
	}

	if v {
		sk.bits |= 0x01 << (7 - sk.bitsCt)
	}
	sk.bitsCt++

	return sk.bitsCt == 8
}

func (sk *SaveKey) popBit() (uint8, bool) {
	if sk.bitsCt >= 8 {
		sk.resetBits()
	}

	if sk.bitsCt == 0 {
		sk.bits = sk.data[sk.address]
		sk.nextAddress()
	}

	v := (sk.bits >> (7 - sk.bitsCt)) & 0x01
	sk.bitsCt++

	if sk.bitsCt >= 8 {
		return v, true
	}

	return v, false
}

func (sk *SaveKey) resetBits() {
	sk.bits = 0
	sk.bitsCt = 0
}

// nextAddress makes sure the address if kept on the same page, by looping back
// to the start of the current page.
func (sk *SaveKey) nextAddress() {
	if sk.address&0x3f == 0x3f {
		sk.address ^= 0x3f
	} else {
		sk.address++
	}
}

func (sk *SaveKey) Step() {
	// update i2c state
	sk.SDA.tick(sk.signal&maskSDA == maskSDA)
	sk.SCL.tick(sk.signal&maskSCL == maskSCL)

	if sk.state > stopped && sk.SCL.hi() && sk.SDA.rising() {
		logger.Log("savekey", "stopped message")
		sk.state = stopped
		sk.writeSaveKey()
		return
	}

	if !sk.SCL.rising() {
		return
	}

	switch sk.ack {
	case ack:
		if sk.dir == reading && sk.SDA.falling() {
			logger.Log("savekey", "nack")
			sk.bus.WriteSWCHx(sk.id, maskSDA)
			sk.ack = none
			return
		}
		logger.Log("savekey", "ack")
		sk.bus.WriteSWCHx(sk.id, 0x00)
		sk.ack = none
		return
	}

	switch sk.state {
	case stopped:
		if sk.SDA.lo() {
			logger.Log("savekey", "starting message")
			sk.resetBits()
			sk.state = starting
		}

	case starting:
		if sk.pushBit(sk.SDA.falling()) {
			switch sk.bits {
			case readSig:
				logger.Log("savekey", "reading message")
				sk.resetBits()
				sk.state = data
				sk.dir = reading
				sk.ack = ack
			case writeSig:
				logger.Log("savekey", "writing message")
				sk.state = addressHi
				sk.dir = writing
				sk.ack = ack
			default:
				logger.Log("savekey", "unrecognised message")
				sk.state = stopped
			}
		}

	case addressHi:
		if sk.pushBit(sk.SDA.falling()) {
			sk.address = uint16(sk.bits) << 8
			sk.state = addressLo
			sk.ack = ack
			logger.Log("savekey", fmt.Sprintf("address hi %#02x", (sk.address>>8)))
		}

	case addressLo:
		if sk.pushBit(sk.SDA.falling()) {
			sk.address |= uint16(sk.bits)
			sk.state = data
			sk.ack = ack
			logger.Log("savekey", fmt.Sprintf("address lo %#02x", sk.address&0xff))

			switch sk.dir {
			case reading:
				logger.Log("savekey", fmt.Sprintf("reading from address %#04x", sk.address))
			case writing:
				logger.Log("savekey", fmt.Sprintf("writing to address %#04x", sk.address))
			}
		}

	case data:
		switch sk.dir {
		case reading:
			v, ok := sk.popBit()

			if v == 0x00 {
				sk.bus.WriteSWCHx(sk.id, 0x00)
			} else {
				sk.bus.WriteSWCHx(sk.id, maskSDA)
			}

			if ok {
				if unicode.IsPrint(rune(sk.bits)) {
					logger.Log("savekey", fmt.Sprintf("read byte %#02x [%c]", sk.bits, sk.bits))
				} else {
					logger.Log("savekey", fmt.Sprintf("read byte %#02x", sk.bits))
				}
				sk.ack = ack
			}

		case writing:
			if sk.pushBit(sk.SDA.falling()) {
				if unicode.IsPrint(rune(sk.bits)) {
					logger.Log("savekey", fmt.Sprintf("written byte %#02x [%c]", sk.bits, sk.bits))
				} else {
					logger.Log("savekey", fmt.Sprintf("written byte %#02x", sk.bits))
				}
				sk.data[sk.address] = sk.bits
				sk.nextAddress()
				sk.ack = ack
			}
		}
	}
}

func (sk *SaveKey) HandleEvent(_ ports.Event, _ ports.EventData) error {
	return nil
}
