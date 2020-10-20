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

// LazyPlayfield lazily accesses playfield information from the emulator.
type LazyPlayfield struct {
	val *LazyValues

	pf              atomic.Value // *video.Playfield
	ctrlpf          atomic.Value // uint8
	foregroundColor atomic.Value // uint8
	backgroundColor atomic.Value // uint8
	reflected       atomic.Value // bool
	scoremode       atomic.Value // bool
	priority        atomic.Value // bool
	region          atomic.Value // video.ScreenRegion
	pf0             atomic.Value // uint8
	pf1             atomic.Value // uint8
	pF2             atomic.Value // uint8
	idx             atomic.Value // int
	leftData        atomic.Value // []bool
	rightData       atomic.Value // []bool

	// Pf is a pointer to the "live" data in the other thread. Do not access
	// the fields in this struct directly. It can be used in PushRawEvent()
	// call
	Pf *video.Playfield

	Ctrlpf          uint8
	ForegroundColor uint8
	BackgroundColor uint8
	Reflected       bool
	Scoremode       bool
	Priority        bool
	Region          video.ScreenRegion
	PF0             uint8
	PF1             uint8
	PF2             uint8
	Idx             int
	LeftData        []bool
	RightData       []bool
}

func newLazyPlayfield(val *LazyValues) *LazyPlayfield {
	return &LazyPlayfield{val: val}
}

func (lz *LazyPlayfield) push() {
	lz.pf.Store(lz.val.Dbg.VCS.TIA.Video.Playfield)
	lz.ctrlpf.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.Ctrlpf)
	lz.foregroundColor.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.ForegroundColor)
	lz.backgroundColor.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.BackgroundColor)
	lz.reflected.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.Reflected)
	lz.scoremode.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.Scoremode)
	lz.priority.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.Priority)
	lz.region.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.Region)
	lz.pf0.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.PF0)
	lz.pf1.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.PF1)
	lz.pF2.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.PF2)
	lz.idx.Store(lz.val.Dbg.VCS.TIA.Video.Playfield.Idx)

	l := make([]bool, video.RegionWidth)
	r := make([]bool, video.RegionWidth)
	copy(l, *lz.val.Dbg.VCS.TIA.Video.Playfield.LeftData)
	copy(r, *lz.val.Dbg.VCS.TIA.Video.Playfield.RightData)
	lz.leftData.Store(l)
	lz.rightData.Store(r)
}

func (lz *LazyPlayfield) update() {
	lz.Pf, _ = lz.pf.Load().(*video.Playfield)
	lz.Ctrlpf, _ = lz.ctrlpf.Load().(uint8)
	lz.ForegroundColor, _ = lz.foregroundColor.Load().(uint8)
	lz.BackgroundColor, _ = lz.backgroundColor.Load().(uint8)
	lz.Reflected, _ = lz.reflected.Load().(bool)
	lz.Scoremode, _ = lz.scoremode.Load().(bool)
	lz.Priority, _ = lz.priority.Load().(bool)
	lz.Region, _ = lz.region.Load().(video.ScreenRegion)
	lz.PF0, _ = lz.pf0.Load().(uint8)
	lz.PF1, _ = lz.pf1.Load().(uint8)
	lz.PF2, _ = lz.pF2.Load().(uint8)
	lz.Idx, _ = lz.idx.Load().(int)
	lz.LeftData, _ = lz.leftData.Load().([]bool)
	lz.RightData, _ = lz.rightData.Load().([]bool)
}
