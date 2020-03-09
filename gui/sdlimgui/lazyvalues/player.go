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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package lazyvalues

import "sync/atomic"

// LazyPlayer lazily accesses player information from the emulator.
type LazyPlayer struct {
	val *Values
	id  int

	atomicResetPixel    atomic.Value // int
	ResetPixel          int
	atomicHmovedPixel   atomic.Value // int
	HmovedPixel         int
	atomicColor         atomic.Value // uint8
	Color               uint8
	atomicNusiz         atomic.Value // uint8
	Nusiz               uint8
	atomicReflected     atomic.Value // bool
	Reflected           bool
	atomicVerticalDelay atomic.Value // bool
	VerticalDelay       bool
	atomicHmove         atomic.Value // uint8
	Hmove               uint8
	atomicMoreHmove     atomic.Value // bool
	MoreHmove           bool
	atomicGfxDataNew    atomic.Value // uint8
	GfxDataNew          uint8
	atomicGfxDataOld    atomic.Value // uint8
	GfxDataOld          uint8

	atomicScanIsActive     atomic.Value // bool
	ScanIsActive           bool
	atomicScanIsLatching   atomic.Value // bool
	ScanIsLatching         bool
	atomicScanPixel        atomic.Value // int
	ScanPixel              int
	atomicScanCpy          atomic.Value // int
	ScanCpy                int
	atomicScanLatchedNusiz atomic.Value // uint8
	ScanLatchedNusiz       uint8
}

func newLazyPlayer(val *Values, id int) *LazyPlayer {
	return &LazyPlayer{val: val, id: id}
}

func (lz *LazyPlayer) update() {
	ps := lz.val.VCS.TIA.Video.Player0
	if lz.id != 0 {
		ps = lz.val.VCS.TIA.Video.Player1
	}
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicResetPixel.Store(ps.ResetPixel)
		lz.atomicHmovedPixel.Store(ps.HmovedPixel)
		lz.atomicColor.Store(ps.Color)
		lz.atomicNusiz.Store(ps.Nusiz)
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
		lz.atomicScanLatchedNusiz.Store(ps.ScanCounter.LatchedNusiz)
	})
	lz.ResetPixel, _ = lz.atomicResetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.atomicHmovedPixel.Load().(int)
	lz.Color, _ = lz.atomicColor.Load().(uint8)
	lz.Nusiz, _ = lz.atomicNusiz.Load().(uint8)
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
	lz.ScanLatchedNusiz, _ = lz.atomicScanLatchedNusiz.Load().(uint8)
}
