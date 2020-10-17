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

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/paths"
)

const saveKeyPath = "savekey"

const eepromSize = 65536

// EEPROM represents the non-volatile memory in the SaveKey peripheral.
type EEPROM struct {
	// the next address an i2c read/write operation will access
	Address uint16

	// whether data is in memory is synced with copy on disk
	Dirty bool

	// amend data only through put() and Poke()
	data []uint8
}

// NewEeprom is the preferred metho of initialisation for the EEPROM type. This
// function will initialise the memory and Read() any existing data from disk.
func newEeprom() *EEPROM {
	ee := &EEPROM{
		data: make([]uint8, eepromSize),
	}

	// initialise data with 0xff
	for i := range ee.data {
		ee.data[i] = 0xff
	}
	ee.Dirty = true

	// load of disk
	ee.Read()

	return ee
}

// Read EEPROM data from disk.
func (ee *EEPROM) Read() {
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
	if fs.Size() != int64(len(ee.data)) {
		logger.Log("savekey", fmt.Sprintf("savekey file is of incorrect length. %d should be 65536 ", fs.Size()))
	}

	_, err = f.Read(ee.data)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not load savekey file (%s)", err))
		return
	}

	logger.Log("savekey", fmt.Sprintf("savekey file loaded from %s", fn))

	// data is synced with disk
	ee.Dirty = false
}

// Write EEPROM data to disk.
func (ee *EEPROM) Write() {
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
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Log("savekey", fmt.Sprintf("could not close savekey file (%s)", err))
		}
	}()

	n, err := f.Write(ee.data)
	if err != nil {
		logger.Log("savekey", fmt.Sprintf("could not write savekey file (%s)", err))
		return
	}

	if n != len(ee.data) {
		logger.Log("savekey", fmt.Sprintf("savekey file has not been truncated during write. %d should be 65536", n))
		return
	}

	logger.Log("savekey", fmt.Sprintf("savekey file saved to %s", fn))

	// data is synced with disk
	ee.Dirty = false
}

// Poke a value into EEPROM.
func (ee *EEPROM) Poke(address uint16, data uint8) {
	ee.data[address] = data
	ee.Dirty = true
}

func (ee *EEPROM) put(v uint8) {
	ee.data[ee.Address] = v
	ee.nextAddress()
	ee.Dirty = true
}

func (ee *EEPROM) get() uint8 {
	defer ee.nextAddress()
	return ee.data[ee.Address]
}

// nextAddress makes sure the address if kept on the same page, by looping back
// to the start of the current page.
func (ee *EEPROM) nextAddress() {
	if ee.Address&0x3f == 0x3f {
		ee.Address ^= 0x3f
	} else {
		ee.Address++
	}
}

// Copy EEPROM data to a new array.
func (ee *EEPROM) Copy() []uint8 {
	d := make([]uint8, len(ee.data))
	copy(d, ee.data)
	return d
}
