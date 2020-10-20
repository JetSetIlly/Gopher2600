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

	"github.com/jetsetilly/gopher2600/hardware/tia/video"
)

// LazyPlayer lazily accesses player information from the emulator.
type LazyPlayer struct {
	val *LazyValues
	id  int

	ps            atomic.Value // *video.PlayerSprite
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

	// P is a pointer to the "live" data in the other thread. Do not access
	// the fields in this struct directly. It can be used in PushRawEvent()
	// call
	Ps *video.PlayerSprite

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
	pl := lz.val.Dbg.VCS.TIA.Video.Player0
	if lz.id != 0 {
		pl = lz.val.Dbg.VCS.TIA.Video.Player1
	}
	lz.ps.Store(pl)
	lz.resetPixel.Store(pl.ResetPixel)
	lz.hmovedPixel.Store(pl.HmovedPixel)
	lz.color.Store(pl.Color)
	lz.nusiz.Store(pl.Nusiz)
	lz.sizeAndCopies.Store(pl.SizeAndCopies)
	lz.reflected.Store(pl.Reflected)
	lz.verticalDelay.Store(pl.VerticalDelay)
	lz.hmove.Store(pl.Hmove)
	lz.moreHmove.Store(pl.MoreHMOVE)
	lz.gfxDataNew.Store(pl.GfxDataNew)
	lz.gfxDataOld.Store(pl.GfxDataOld)
	lz.scanIsActive.Store(pl.ScanCounter.IsActive())
	lz.scanIsLatching.Store(pl.ScanCounter.IsLatching())
	lz.scanPixel.Store(pl.ScanCounter.Pixel)
	lz.scanCpy.Store(pl.ScanCounter.Cpy)
	lz.scanLatchedSizeAndCopies.Store(pl.ScanCounter.LatchedSizeAndCopies)
}

func (lz *LazyPlayer) update() {
	lz.Ps, _ = lz.ps.Load().(*video.PlayerSprite)
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
