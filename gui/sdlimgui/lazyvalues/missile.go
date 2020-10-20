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

// LazyMissile lazily accesses missile information from the emulator.
type LazyMissile struct {
	val *LazyValues
	id  int

	ms            atomic.Value // *video.MissileSprite
	resetPixel    atomic.Value // int
	hmovedPixel   atomic.Value // int
	color         atomic.Value // uint8
	enabled       atomic.Value // bool
	nusiz         atomic.Value // uint8
	size          atomic.Value // uint8
	copies        atomic.Value // uint8
	hmove         atomic.Value // uint8
	moreHmove     atomic.Value // bool
	resetToPlayer atomic.Value // bool

	Ms            *video.MissileSprite
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

	encActive     atomic.Value // bool
	encSecondHalf atomic.Value // bool
	encCpy        atomic.Value // int
	encTicks      atomic.Value // int

	EncActive     bool
	EncSecondHalf bool
	EncCpy        int
	EncTicks      int
}

func newLazyMissile(val *LazyValues, id int) *LazyMissile {
	return &LazyMissile{val: val, id: id}
}

func (lz *LazyMissile) push() {
	ms := lz.val.Dbg.VCS.TIA.Video.Missile0
	if lz.id != 0 {
		ms = lz.val.Dbg.VCS.TIA.Video.Missile1
	}
	lz.ms.Store(ms)
	lz.resetPixel.Store(ms.ResetPixel)
	lz.hmovedPixel.Store(ms.HmovedPixel)
	lz.color.Store(ms.Color)
	lz.enabled.Store(ms.Enabled)
	lz.nusiz.Store(ms.Nusiz)
	lz.size.Store(ms.Size)
	lz.copies.Store(ms.Copies)
	lz.hmove.Store(ms.Hmove)
	lz.moreHmove.Store(ms.MoreHMOVE)
	lz.resetToPlayer.Store(ms.ResetToPlayer)
	lz.encActive.Store(ms.Enclockifier.Active)
	lz.encSecondHalf.Store(ms.Enclockifier.SecondHalf)
	lz.encCpy.Store(ms.Enclockifier.Cpy)
	lz.encTicks.Store(ms.Enclockifier.Ticks)
}

func (lz *LazyMissile) update() {
	lz.Ms, _ = lz.ms.Load().(*video.MissileSprite)
	lz.ResetPixel, _ = lz.resetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.hmovedPixel.Load().(int)
	lz.Color, _ = lz.color.Load().(uint8)
	lz.Enabled, _ = lz.enabled.Load().(bool)
	lz.Nusiz, _ = lz.nusiz.Load().(uint8)
	lz.Size, _ = lz.size.Load().(uint8)
	lz.Copies, _ = lz.copies.Load().(uint8)
	lz.Hmove, _ = lz.hmove.Load().(uint8)
	lz.MoreHmove, _ = lz.moreHmove.Load().(bool)
	lz.ResetToPlayer, _ = lz.resetToPlayer.Load().(bool)
	lz.EncActive, _ = lz.encActive.Load().(bool)
	lz.EncSecondHalf, _ = lz.encSecondHalf.Load().(bool)
	lz.EncCpy, _ = lz.encCpy.Load().(int)
	lz.EncTicks, _ = lz.encTicks.Load().(int)
}
