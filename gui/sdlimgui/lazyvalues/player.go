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

// LazyPlayer lazily accesses player information from the emulator
type LazyPlayer struct {
	val *Lazy
	id  int

	atomicResetPixel    atomic.Value // int
	atomicHmovedPixel   atomic.Value // int
	atomicColor         atomic.Value // uint8
	atomicNusiz         atomic.Value // uint8
	atomicSizeAndCopies atomic.Value // uint8
	atomicReflected     atomic.Value // bool
	atomicVerticalDelay atomic.Value // bool
	atomicHmove         atomic.Value // uint8
	atomicMoreHmove     atomic.Value // bool
	atomicGfxDataNew    atomic.Value // uint8
	atomicGfxDataOld    atomic.Value // uint8

	ResetPixel    int
	HmovedPixel   int
	Color         uint8
	Nusiz         uint8
	SizeAndCopies uint8
	Reflected     bool
	VerticalDelay bool
	Hmove         uint8
	MoreHmove     bool
	GfxDataNew    uint8
	GfxDataOld    uint8

	atomicScanIsActive             atomic.Value // bool
	atomicScanIsLatching           atomic.Value // bool
	atomicScanPixel                atomic.Value // int
	atomicScanCpy                  atomic.Value // int
	atomicScanLatchedSizeAndCopies atomic.Value // uint8

	ScanIsActive             bool
	ScanIsLatching           bool
	ScanPixel                int
	ScanCpy                  int
	ScanLatchedSizeAndCopies uint8
}

func newLazyPlayer(val *Lazy, id int) *LazyPlayer {
	return &LazyPlayer{val: val, id: id}
}

func (lz *LazyPlayer) update() {
	ps := lz.val.Dbg.VCS.TIA.Video.Player0
	if lz.id != 0 {
		ps = lz.val.Dbg.VCS.TIA.Video.Player1
	}
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicResetPixel.Store(ps.ResetPixel)
		lz.atomicHmovedPixel.Store(ps.HmovedPixel)
		lz.atomicColor.Store(ps.Color)
		lz.atomicNusiz.Store(ps.Nusiz)
		lz.atomicSizeAndCopies.Store(ps.SizeAndCopies)
		lz.atomicReflected.Store(ps.Reflected)
		lz.atomicVerticalDelay.Store(ps.VerticalDelay)
		lz.atomicHmove.Store(ps.Hmove)
		lz.atomicMoreHmove.Store(ps.MoreHMOVE)
		lz.atomicGfxDataNew.Store(ps.GfxDataNew)
		lz.atomicGfxDataOld.Store(ps.GfxDataOld)
		lz.atomicScanIsActive.Store(ps.ScanCounter.IsActive())
		lz.atomicScanIsLatching.Store(ps.ScanCounter.IsLatching())
		lz.atomicScanPixel.Store(ps.ScanCounter.Pixel)
		lz.atomicScanCpy.Store(ps.ScanCounter.Cpy)
		lz.atomicScanLatchedSizeAndCopies.Store(ps.ScanCounter.LatchedSizeAndCopies)
	})
	lz.ResetPixel, _ = lz.atomicResetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.atomicHmovedPixel.Load().(int)
	lz.Color, _ = lz.atomicColor.Load().(uint8)
	lz.Nusiz, _ = lz.atomicNusiz.Load().(uint8)
	lz.SizeAndCopies, _ = lz.atomicSizeAndCopies.Load().(uint8)
	lz.Reflected, _ = lz.atomicReflected.Load().(bool)
	lz.VerticalDelay, _ = lz.atomicVerticalDelay.Load().(bool)
	lz.Hmove, _ = lz.atomicHmove.Load().(uint8)
	lz.MoreHmove, _ = lz.atomicMoreHmove.Load().(bool)
	lz.GfxDataNew, _ = lz.atomicGfxDataNew.Load().(uint8)
	lz.GfxDataOld, _ = lz.atomicGfxDataOld.Load().(uint8)
	lz.ScanIsActive, _ = lz.atomicScanIsActive.Load().(bool)
	lz.ScanIsLatching, _ = lz.atomicScanIsLatching.Load().(bool)
	lz.ScanPixel, _ = lz.atomicScanPixel.Load().(int)
	lz.ScanCpy, _ = lz.atomicScanCpy.Load().(int)
	lz.ScanLatchedSizeAndCopies, _ = lz.atomicScanLatchedSizeAndCopies.Load().(uint8)
}
