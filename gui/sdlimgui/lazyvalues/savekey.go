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

package lazyvalues

import (
	"sync/atomic"

	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
)

// LazyChipRegisters lazily accesses chip registere information from the emulator.
type LazySaveKey struct {
	val *LazyValues

	saveKeyActive atomic.Value // bool
	SaveKeyActive bool

	sda        atomic.Value // []float32
	scl        atomic.Value // []float32
	state      atomic.Value // savekey.MessageState
	dir        atomic.Value // savekey.DataDirection
	ack        atomic.Value // bool
	bits       atomic.Value // uint8
	bitsCt     atomic.Value // int
	address    atomic.Value // uint16
	eepromData atomic.Value // []uint8
	dirty      atomic.Value // bool

	SDA        []float32
	SCL        []float32
	State      savekey.MessageState
	Dir        savekey.DataDirection
	Ack        bool
	Bits       uint8
	BitsCt     int
	Address    uint16
	EEPROMdata []uint8
	Dirty      bool
}

func newLazySaveKey(val *LazyValues) *LazySaveKey {
	return &LazySaveKey{val: val}
}

func (lz *LazySaveKey) push() {
	if sk, ok := lz.val.Dbg.VCS.RIOT.Ports.Player1.(*savekey.SaveKey); ok {
		lz.saveKeyActive.Store(true)
		lz.sda.Store(sk.SDA.Copy())
		lz.scl.Store(sk.SCL.Copy())
		lz.state.Store(sk.State)
		lz.dir.Store(sk.Dir)
		lz.ack.Store(sk.Ack)
		lz.bits.Store(sk.Bits)
		lz.bitsCt.Store(sk.BitsCt)
		lz.address.Store(sk.EEPROM.Address)
		lz.eepromData.Store(sk.EEPROM.Copy())
		lz.dirty.Store(sk.EEPROM.Dirty)
	} else {
		lz.saveKeyActive.Store(false)
	}
}

func (lz *LazySaveKey) update() {
	if l, ok := lz.saveKeyActive.Load().(bool); l && ok {
		lz.SaveKeyActive = true
	} else {
		lz.SaveKeyActive = false
		return
	}

	if l, ok := lz.sda.Load().([]float32); ok {
		lz.SDA = l
	}

	if l, ok := lz.scl.Load().([]float32); ok {
		lz.SCL = l
	}

	lz.State = lz.state.Load().(savekey.MessageState)
	lz.Dir = lz.dir.Load().(savekey.DataDirection)
	lz.Ack = lz.ack.Load().(bool)
	lz.Bits = lz.bits.Load().(uint8)
	lz.BitsCt = lz.bitsCt.Load().(int)
	lz.Address = lz.address.Load().(uint16)

	if l, ok := lz.eepromData.Load().([]uint8); ok {
		lz.EEPROMdata = l
	}

	lz.Dirty = lz.dirty.Load().(bool)
}
