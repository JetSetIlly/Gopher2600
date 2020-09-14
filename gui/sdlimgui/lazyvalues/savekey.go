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

// LazyChipRegisters lazily accesses chip registere information from the emulator
type LazySaveKey struct {
	val *Lazy

	atomicSaveKeyActive atomic.Value // bool
	SaveKeyActive       bool

	atomicSDA        atomic.Value // []float32
	atomicSCL        atomic.Value // []float32
	atomicState      atomic.Value // savekey.MessageState
	atomicDir        atomic.Value // savekey.DataDirection
	atomicAck        atomic.Value // bool
	atomicBits       atomic.Value // uint8
	atomicBitsCt     atomic.Value // int
	atomicAddress    atomic.Value // uint16
	atomicEEPROMdata atomic.Value // []uint8
	atomicDirty      atomic.Value // bool
	SDA              []float32
	SCL              []float32
	State            savekey.MessageState
	Dir              savekey.DataDirection
	Ack              bool
	Bits             uint8
	BitsCt           int
	Address          uint16
	EEPROMdata       []uint8
	Dirty            bool
}

func newLazySaveKey(val *Lazy) *LazySaveKey {
	return &LazySaveKey{val: val}
}

func (lz *LazySaveKey) update() {
	lz.val.Dbg.PushRawEvent(func() {
		if sk, ok := lz.val.Dbg.VCS.RIOT.Ports.Player1.(*savekey.SaveKey); ok {
			lz.atomicSaveKeyActive.Store(true)
			lz.atomicSDA.Store(sk.SDA.Copy())
			lz.atomicSCL.Store(sk.SCL.Copy())
			lz.atomicState.Store(sk.State)
			lz.atomicDir.Store(sk.Dir)
			lz.atomicAck.Store(sk.Ack)
			lz.atomicBits.Store(sk.Bits)
			lz.atomicBitsCt.Store(sk.BitsCt)
			lz.atomicAddress.Store(sk.EEPROM.Address)
			lz.atomicEEPROMdata.Store(sk.EEPROM.Copy())
			lz.atomicDirty.Store(sk.EEPROM.Dirty)
		} else {
			lz.atomicSaveKeyActive.Store(false)
		}
	})

	if l, ok := lz.atomicSaveKeyActive.Load().(bool); l && ok {
		lz.SaveKeyActive = true
	} else {
		lz.SaveKeyActive = false
		return
	}

	if l, ok := lz.atomicSDA.Load().([]float32); ok {
		lz.SDA = l
	}

	if l, ok := lz.atomicSCL.Load().([]float32); ok {
		lz.SCL = l
	}

	lz.State = lz.atomicState.Load().(savekey.MessageState)
	lz.Dir = lz.atomicDir.Load().(savekey.DataDirection)
	lz.Ack = lz.atomicAck.Load().(bool)
	lz.Bits = lz.atomicBits.Load().(uint8)
	lz.BitsCt = lz.atomicBitsCt.Load().(int)
	lz.Address = lz.atomicAddress.Load().(uint16)

	if l, ok := lz.atomicEEPROMdata.Load().([]uint8); ok {
		lz.EEPROMdata = l
	}

	lz.Dirty = lz.atomicDirty.Load().(bool)
}
