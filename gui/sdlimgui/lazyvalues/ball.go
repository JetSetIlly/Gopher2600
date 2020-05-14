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

// LazyBall lazily accesses ball information from the emulator
type LazyBall struct {
	val *Lazy

	atomicResetPixel    atomic.Value // int
	atomicHmovedPixel   atomic.Value // int
	atomicColor         atomic.Value // uint8
	atomicVerticalDelay atomic.Value // bool
	atomicEnabledDelay  atomic.Value // bool
	atomicEnabled       atomic.Value // bool
	atomicCtrlpf        atomic.Value // uint8
	atomicSize          atomic.Value // uint8
	atomicHmove         atomic.Value // uint8
	atomicMoreHmove     atomic.Value // bool

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

	atomicEncActive     atomic.Value // bool
	atomicEncSecondHalf atomic.Value // bool
	atomicEncCpy        atomic.Value // int

	EncActive     bool
	EncSecondHalf bool
	EncCpy        int
}

func newLazyBall(val *Lazy) *LazyBall {
	return &LazyBall{val: val}
}

func (lz *LazyBall) update() {
	lz.val.Dbg.PushRawEvent(func() {
		lz.atomicResetPixel.Store(lz.val.VCS.TIA.Video.Ball.ResetPixel)
		lz.atomicHmovedPixel.Store(lz.val.VCS.TIA.Video.Ball.HmovedPixel)
		lz.atomicColor.Store(lz.val.VCS.TIA.Video.Ball.Color)
		lz.atomicVerticalDelay.Store(lz.val.VCS.TIA.Video.Ball.VerticalDelay)
		lz.atomicEnabledDelay.Store(lz.val.VCS.TIA.Video.Ball.EnabledDelay)
		lz.atomicEnabled.Store(lz.val.VCS.TIA.Video.Ball.Enabled)
		lz.atomicCtrlpf.Store(lz.val.VCS.TIA.Video.Ball.Ctrlpf)
		lz.atomicSize.Store(lz.val.VCS.TIA.Video.Ball.Size)
		lz.atomicHmove.Store(lz.val.VCS.TIA.Video.Ball.Hmove)
		lz.atomicMoreHmove.Store(lz.val.VCS.TIA.Video.Ball.MoreHMOVE)
		lz.atomicEncActive.Store(lz.val.VCS.TIA.Video.Ball.Enclockifier.Active)
		lz.atomicEncSecondHalf.Store(lz.val.VCS.TIA.Video.Ball.Enclockifier.SecondHalf)
		lz.atomicEncCpy.Store(lz.val.VCS.TIA.Video.Ball.Enclockifier.Cpy)
	})
	lz.ResetPixel, _ = lz.atomicResetPixel.Load().(int)
	lz.HmovedPixel, _ = lz.atomicHmovedPixel.Load().(int)
	lz.Color, _ = lz.atomicColor.Load().(uint8)
	lz.VerticalDelay, _ = lz.atomicVerticalDelay.Load().(bool)
	lz.EnabledDelay, _ = lz.atomicEnabledDelay.Load().(bool)
	lz.Enabled, _ = lz.atomicEnabled.Load().(bool)
	lz.Ctrlpf, _ = lz.atomicCtrlpf.Load().(uint8)
	lz.Size, _ = lz.atomicSize.Load().(uint8)
	lz.Hmove, _ = lz.atomicHmove.Load().(uint8)
	lz.MoreHmove, _ = lz.atomicMoreHmove.Load().(bool)
	lz.EncActive, _ = lz.atomicEncActive.Load().(bool)
	lz.EncSecondHalf, _ = lz.atomicEncSecondHalf.Load().(bool)
	lz.EncCpy, _ = lz.atomicEncCpy.Load().(int)
}
