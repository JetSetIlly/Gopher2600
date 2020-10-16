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

// LazyPlayer lazily accesses player information from the emulator.
type LazyPlayer struct {
	val *LazyValues
	id  int

	resetPixel    atomic.Value // int
	hmovedPixel   atomic.Value // int
	color         atomic.Value // uint8
	nusiz         atomic.Value // uint8
	sizeAndCopies atomic.Value // uint8
	reflected     atomic.Value // bool
	verticalDelay atomic.Value // bool
	hmove         atomic.Value // uint8
	moreHmove     atomic.Value // bool
	gfxDataNew    atomic.Value // uint8
	gfxDataOld    atomic.Value // uint8

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

	scanIsActive             atomic.Value // bool
	scanIsLatching           atomic.Value // bool
	scanPixel                atomic.Value // int
	scanCpy                  atomic.Value // int
	scanLatchedSizeAndCopies atomic.Value // uint8

	ScanIsActive             bool
	ScanIsLatching           bool
	ScanPixel                int
	ScanCpy                  int
	ScanLatchedSizeAndCopies uint8
}

func newLazyPlayer(val *LazyValues, id int) *LazyPlayer {
	return &LazyPlayer{val: val, id: id}
}

func (lz *LazyPlayer) push() {
	ps := lz.val.Dbg.VCS.TIA.Video.Player0
	if lz.id != 0 {
		ps = lz.val.Dbg.VCS.TIA.Video.Player1
	}
	lz.resetPixel.Store(ps.ResetPixel)
	lz.hmovedPixel.Store(ps.HmovedPixel)
	lz.color.Store(ps.Color)
	lz.nusiz.Store(ps.Nusiz)
	lz.sizeAndCopies.Store(ps.SizeAndCopies)
	lz.reflected.Store(ps.Reflected)
	lz.verticalDelay.Store(ps.VerticalDelay)
	lz.hmove.Store(ps.Hmove)
	lz.moreHmove.Store(ps.MoreHMOVE)
	lz.gfxDataNew.Store(ps.GfxDataNew)
	lz.gfxDataOld.Store(ps.GfxDataOld)
	lz.scanIsActive.Store(ps.ScanCounter.IsActive())
	lz.scanIsLatching.Store(ps.ScanCounter.IsLatching())
	lz.scanPixel.Store(ps.ScanCounter.Pixel)
	lz.scanCpy.Store(ps.ScanCounter.Cpy)
	lz.scanLatchedSizeAndCopies.Store(ps.ScanCounter.LatchedSizeAndCopies)
}

func (lz *LazyPlayer) update() {
	lz.ResetPixel, _ = lz.resetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.hmovedPixel.Load().(int)
	lz.Color, _ = lz.color.Load().(uint8)
	lz.Nusiz, _ = lz.nusiz.Load().(uint8)
	lz.SizeAndCopies, _ = lz.sizeAndCopies.Load().(uint8)
	lz.Reflected, _ = lz.reflected.Load().(bool)
	lz.VerticalDelay, _ = lz.verticalDelay.Load().(bool)
	lz.Hmove, _ = lz.hmove.Load().(uint8)
	lz.MoreHmove, _ = lz.moreHmove.Load().(bool)
	lz.GfxDataNew, _ = lz.gfxDataNew.Load().(uint8)
	lz.GfxDataOld, _ = lz.gfxDataOld.Load().(uint8)
	lz.ScanIsActive, _ = lz.scanIsActive.Load().(bool)
	lz.ScanIsLatching, _ = lz.scanIsLatching.Load().(bool)
	lz.ScanPixel, _ = lz.scanPixel.Load().(int)
	lz.ScanCpy, _ = lz.scanCpy.Load().(int)
	lz.ScanLatchedSizeAndCopies, _ = lz.scanLatchedSizeAndCopies.Load().(uint8)
}
