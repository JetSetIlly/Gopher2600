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

	SaveKeyActive bool

	atomicSDA atomic.Value // []float32
	atomicSCL atomic.Value // []float32
	SDA       []float32
	SCL       []float32
}

func newLazySaveKey(val *Lazy) *LazySaveKey {
	return &LazySaveKey{val: val}
}

func (lz *LazySaveKey) update() {
	lz.val.Dbg.PushRawEvent(func() {
		if l, ok := lz.val.Dbg.VCS.RIOT.Ports.Player1.(*savekey.SaveKey); ok {
			lz.atomicSDA.Store(l.SDA.Copy())
			lz.atomicSCL.Store(l.SCL.Copy())
		}
	})

	if l, ok := lz.atomicSDA.Load().([]float32); ok {
		lz.SaveKeyActive = ok
		lz.SDA = l
	}

	if l, ok := lz.atomicSCL.Load().([]float32); ok {
		lz.SaveKeyActive = ok
		lz.SCL = l
	}
}
