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

// LazyBall lazily accesses ball information from the emulator.
type LazyBall struct {
	val *LazyValues

	bs            atomic.Value // *video.BallSprite
	resetPixel    atomic.Value // int
	hmovedPixel   atomic.Value // int
	color         atomic.Value // uint8
	verticalDelay atomic.Value // bool
	enabledDelay  atomic.Value // bool
	enabled       atomic.Value // bool
	ctrlpf        atomic.Value // uint8
	size          atomic.Value // uint8
	hmove         atomic.Value // uint8
	moreHmove     atomic.Value // bool

	// Bs is a pointer to the "live" data in the other thread. Do not access
	// the fields in this struct directly. It can be used in PushRawEvent()
	// call
	Bs *video.BallSprite

	ResetPixel    int
	HmovedPixel   int
	Color         uint8
	VerticalDelay bool
	EnabledDelay  bool
	Enabled       bool
	Ctrlpf        uint8
	Size          uint8
	Hmove         uint8
	MoreHmove     bool

	encActive     atomic.Value // bool
	encSecondHalf atomic.Value // bool
	encCpy        atomic.Value // int
	encTicks      atomic.Value // int

	EncActive     bool
	EncSecondHalf bool
	EncCpy        int
	EncTicks      int
}

func newLazyBall(val *LazyValues) *LazyBall {
	return &LazyBall{val: val}
}

func (lz *LazyBall) push() {
	lz.bs.Store(lz.val.Dbg.VCS.TIA.Video.Ball)
	lz.resetPixel.Store(lz.val.Dbg.VCS.TIA.Video.Ball.ResetPixel)
	lz.hmovedPixel.Store(lz.val.Dbg.VCS.TIA.Video.Ball.HmovedPixel)
	lz.color.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Color)
	lz.verticalDelay.Store(lz.val.Dbg.VCS.TIA.Video.Ball.VerticalDelay)
	lz.enabledDelay.Store(lz.val.Dbg.VCS.TIA.Video.Ball.EnabledDelay)
	lz.enabled.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Enabled)
	lz.ctrlpf.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Ctrlpf)
	lz.size.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Size)
	lz.hmove.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Hmove)
	lz.moreHmove.Store(lz.val.Dbg.VCS.TIA.Video.Ball.MoreHMOVE)
	lz.encActive.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Enclockifier.Active)
	lz.encSecondHalf.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Enclockifier.SecondHalf)
	lz.encCpy.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Enclockifier.Cpy)
	lz.encTicks.Store(lz.val.Dbg.VCS.TIA.Video.Ball.Enclockifier.Ticks)
}

func (lz *LazyBall) update() {
	lz.Bs, _ = lz.bs.Load().(*video.BallSprite)
	lz.ResetPixel, _ = lz.resetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.hmovedPixel.Load().(int)
	lz.Color, _ = lz.color.Load().(uint8)
	lz.VerticalDelay, _ = lz.verticalDelay.Load().(bool)
	lz.EnabledDelay, _ = lz.enabledDelay.Load().(bool)
	lz.Enabled, _ = lz.enabled.Load().(bool)
	lz.Ctrlpf, _ = lz.ctrlpf.Load().(uint8)
	lz.Size, _ = lz.size.Load().(uint8)
	lz.Hmove, _ = lz.hmove.Load().(uint8)
	lz.MoreHmove, _ = lz.moreHmove.Load().(bool)
	lz.EncActive, _ = lz.encActive.Load().(bool)
	lz.EncSecondHalf, _ = lz.encSecondHalf.Load().(bool)
	lz.EncCpy, _ = lz.encCpy.Load().(int)
	lz.EncTicks, _ = lz.encTicks.Load().(int)
}
