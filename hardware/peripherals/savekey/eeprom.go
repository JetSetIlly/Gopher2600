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
	"os"
	"slices"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources"
)

const saveKeyPath = "savekey"

const (
	EEPROMsize     = 0x8000
	EEPROMpageSize = 0x40
	EEPROMnumPages = EEPROMsize / EEPROMpageSize
)

// EEPROM represents the non-volatile memory in the SaveKey peripheral.
type EEPROM struct {
	env *environment.Environment

	// the next address an i2c read/write operation will access
	Address uint16

	// amend Data only through put() and Poke()
	Data []uint8

	// the data as it is on disk. data is mutable and we need a way of
	// comparing what's on disk with what's in memory.
	DiskData []uint8

	// whether a page has been accessed
	PageAccess []bool
}

// NewEeprom is the preferred metho of initialisation for the EEPROM type. This
// function will initialise the memory and Read() any existing data from disk.
func newEeprom(env *environment.Environment) *EEPROM {
	ee := &EEPROM{
		env:        env,
		Data:       make([]uint8, EEPROMsize),
		DiskData:   make([]uint8, EEPROMsize),
		PageAccess: make([]bool, EEPROMnumPages),
	}

	// initialise data with 0xff
	for i := range ee.Data {
		ee.Data[i] = 0xff
	}

	// load of disk
	ee.Read()

	return ee
}

func (ee *EEPROM) reset() {
	clear(ee.PageAccess)
	ee.Read()
}

func (ee *EEPROM) snapshot() *EEPROM {
	cp := *ee
	cp.Data = make([]uint8, len(ee.Data))
	cp.DiskData = make([]uint8, len(ee.DiskData))
	cp.PageAccess = make([]bool, len(ee.PageAccess))
	copy(cp.Data, ee.Data)
	copy(cp.DiskData, ee.DiskData)
	copy(cp.PageAccess, ee.PageAccess)
	return &cp
}

func (ee *EEPROM) plumb() {
	// TODO: ee.DiskData should really be a mirror of what's on disk so we should save ee.DiskData
	// on plumb. the problem is that plumb() can be called quite often so we really need a scheduler
	// that is reset on every call to plumb() and then only save after a safe period
}

// Read EEPROM data from disk.
func (ee *EEPROM) Read() {
	fn, err := resources.JoinPath(saveKeyPath)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not load eeprom file: %v", err)
		return
	}

	f, err := os.Open(fn)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not load eeprom file: %v", err)
		return
	}
	defer f.Close()

	// get file info. not using Stat() on the file handle because the
	// windows version (when running under wine) does not handle that
	fs, err := os.Stat(fn)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not load eeprom file: %v", err)
		return
	}
	if fs.Size() != int64(len(ee.Data)) {
		logger.Logf(ee.env, "savekey", "eeprom file is of incorrect length. %d should be 65536 ", fs.Size())
	}

	_, err = f.Read(ee.Data)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not load eeprom file: %v", err)
		return
	}

	// copy of data read from disk
	copy(ee.DiskData, ee.Data)

	logger.Logf(ee.env, "savekey", "eeprom file loaded from %s", fn)
}

// Write EEPROM data to disk.
func (ee *EEPROM) Write() {
	fn, err := resources.JoinPath(saveKeyPath)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not write eeprom file: %v", err)
		return
	}

	f, err := os.Create(fn)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not write eeprom file: %v", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Logf(ee.env, "savekey", "could not close eeprom file: %v", err)
		}
	}()

	n, err := f.Write(ee.Data)
	if err != nil {
		logger.Logf(ee.env, "savekey", "could not write eeprom file: %v", err)
		return
	}

	if n != len(ee.Data) {
		logger.Logf(ee.env, "savekey", "eeprom file has not been truncated during write. %d should be 65536", n)
		return
	}

	logger.Logf(ee.env, "savekey", "eeprom file saved to %s", fn)

	// copy of data that's just been written to disk
	copy(ee.DiskData, ee.Data)
}

// Poke a value into EEPROM.
func (ee *EEPROM) Poke(address uint16, data uint8) {
	ee.Data[address] = data
}

func (ee *EEPROM) access() {
	p := ee.Address / EEPROMpageSize
	ee.PageAccess[p] = true
}

func (ee *EEPROM) put(v uint8) {
	ee.access()
	ee.Data[ee.Address] = v
	ee.nextAddress()
}

func (ee *EEPROM) get() uint8 {
	defer func() {
		ee.nextAddress()
		ee.access()
	}()
	return ee.Data[ee.Address]
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

// IsSaved returns true if disk data is the same as data
func (ee *EEPROM) IsSaved() bool {
	return slices.Compare(ee.Data, ee.DiskData) == 0
}
