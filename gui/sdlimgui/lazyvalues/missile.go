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

import "sync/atomic"

// LazyMissile lazily accesses missile information from the emulator
type LazyMissile struct {
	val *Lazy
	id  int

	atomicResetPixel    atomic.Value // int
	atomicHmovedPixel   atomic.Value // int
	atomicColor         atomic.Value // uint8
	atomicEnabled       atomic.Value // bool
	atomicNusiz         atomic.Value // uint8
	atomicSize          atomic.Value // uint8
	atomicCopies        atomic.Value // uint8
	atomicHmove         atomic.Value // uint8
	atomicMoreHmove     atomic.Value // bool
	atomicResetToPlayer atomic.Value // bool

	ResetPixel    int
	HmovedPixel   int
	Color         uint8
	Enabled       bool
	Nusiz         uint8
	Size          uint8
	Copies        uint8
	Hmove         uint8
	MoreHmove     bool
	ResetToPlayer bool

	atomicEncActive     atomic.Value // bool
	atomicEncSecondHalf atomic.Value // bool
	atomicEncCpy        atomic.Value // int
	atomicEncTicks      atomic.Value // int

	EncActive     bool
	EncSecondHalf bool
	EncCpy        int
	EncTicks      int
}

func newLazyMissile(val *Lazy, id int) *LazyMissile {
	return &LazyMissile{val: val, id: id}
}

func (lz *LazyMissile) update() {
	ms := lz.val.Dbg.VCS.TIA.Video.Missile0
	if lz.id != 0 {
		ms = lz.val.Dbg.VCS.TIA.Video.Missile1
	}
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicResetPixel.Store(ms.ResetPixel)
		lz.atomicHmovedPixel.Store(ms.HmovedPixel)
		lz.atomicColor.Store(ms.Color)
		lz.atomicEnabled.Store(ms.Enabled)
		lz.atomicNusiz.Store(ms.Nusiz)
		lz.atomicSize.Store(ms.Size)
		lz.atomicCopies.Store(ms.Copies)
		lz.atomicHmove.Store(ms.Hmove)
		lz.atomicMoreHmove.Store(ms.MoreHMOVE)
		lz.atomicResetToPlayer.Store(ms.ResetToPlayer)
		lz.atomicEncActive.Store(ms.Enclockifier.Active)
		lz.atomicEncSecondHalf.Store(ms.Enclockifier.SecondHalf)
		lz.atomicEncCpy.Store(ms.Enclockifier.Cpy)
		lz.atomicEncTicks.Store(ms.Enclockifier.Ticks)
	})
	lz.ResetPixel, _ = lz.atomicResetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.atomicHmovedPixel.Load().(int)
	lz.Color, _ = lz.atomicColor.Load().(uint8)
	lz.Enabled, _ = lz.atomicEnabled.Load().(bool)
	lz.Nusiz, _ = lz.atomicNusiz.Load().(uint8)
	lz.Size, _ = lz.atomicSize.Load().(uint8)
	lz.Copies, _ = lz.atomicCopies.Load().(uint8)
	lz.Hmove, _ = lz.atomicHmove.Load().(uint8)
	lz.MoreHmove, _ = lz.atomicMoreHmove.Load().(bool)
	lz.ResetToPlayer, _ = lz.atomicResetToPlayer.Load().(bool)
	lz.EncActive, _ = lz.atomicEncActive.Load().(bool)
	lz.EncSecondHalf, _ = lz.atomicEncSecondHalf.Load().(bool)
	lz.EncCpy, _ = lz.atomicEncCpy.Load().(int)
	lz.EncTicks, _ = lz.atomicEncTicks.Load().(int)
}
