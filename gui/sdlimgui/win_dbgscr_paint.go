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

package sdlimgui

import (
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/reflection"
)

func (win *winDbgScr) paintDragAndDrop() {
	// each datastream image can also be dropped onto
	if imgui.BeginDragDropTarget() {
		// drag and drop has ended on a legitimate target
		payload := imgui.AcceptDragDropPayload(painDragDrop, imgui.DragDropFlagsNone)
		if payload != nil {
			mouse := win.currentMouse()
			if mouse.valid {
				ref := win.scr.crit.reflection[mouse.offset]

				ff := floodFill{
					win:  win,
					from: uint8(ref.Signal.Color) & 0xfe,
					to:   payload[0] & 0xfe,
				}

				if ff.from != ff.to {
					ff.startingReflection = make([]reflection.ReflectedVideoStep, len(win.scr.crit.reflection))
					copy(ff.startingReflection, win.scr.crit.reflection)

					target := floodFillTarget{}
					target.coord = win.img.cache.TV.GetCoords()
					target.coord.Clock = mouse.tv.Clock
					target.coord.Scanline = mouse.tv.Scanline
					target.offset = mouse.offset
					ff.build(target)

					logger.Logf(logger.Allow, "paint", "flood fill starting at %s", target.coord)
					ff.resolve(0)
				}
			}
		}
		imgui.EndDragDropTarget()
	}
}

type floodFillTarget struct {
	coord  coords.TelevisionCoords
	offset int
}

type floodFill struct {
	win *winDbgScr

	startingReflection []reflection.ReflectedVideoStep
	targets            []floodFillTarget

	from uint8
	to   uint8
}

func (ff *floodFill) build(target floodFillTarget) {
	ref := ff.startingReflection[target.offset]
	if ref.Signal.VBlank {
		return
	}

	px := uint8(ref.Signal.Color) & 0xfe
	if px != ff.from {
		return
	}

	ff.targets = append(ff.targets, target)
	ref.Signal.Color = signal.ColorSignal(ff.to)
	ff.startingReflection[target.offset] = ref

	// down
	down := target
	down.offset += specification.ClksScanline
	down.coord.Scanline++
	if down.coord.Scanline <= ff.win.img.cache.TV.GetFrameInfo().TotalScanlines {
		ff.build(down)
	}

	// up
	up := target
	up.offset -= specification.ClksScanline
	up.coord.Scanline--
	if up.coord.Scanline >= 0 {
		ff.build(up)
	}

	// left
	left := target
	left.offset--
	left.coord.Clock--
	if left.coord.Clock >= 0 {
		ff.build(left)
	}

	// right
	right := target
	right.offset++
	right.coord.Clock++
	if right.coord.Clock <= specification.ClksScanline {
		ff.build(right)
	}
}

func (ff *floodFill) resolve(offsetIdx int) {
	if offsetIdx >= len(ff.targets) {
		return
	}

	ff.win.img.dbg.PushFunctionImmediate(func() {
		target := ff.targets[offsetIdx]
		ref := ff.win.scr.crit.reflection[target.offset]
		px := uint8(ref.Signal.Color) & 0xfe
		if px != ff.from {
			ff.resolve(offsetIdx + 1)
			return
		}
		ff.win.img.dbg.GotoCoords(target.coord)

		ff.win.img.dbg.PushFunctionImmediate(func() {
			switch ref.VideoElement {
			case video.ElementBackground:
				ff.win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUBK], px, ff.to, 0xfe, func() { ff.resolve(offsetIdx + 1) })
			case video.ElementPlayfield:
				fallthrough
			case video.ElementBall:
				ff.win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUPF], px, ff.to, 0xfe, func() { ff.resolve(offsetIdx + 1) })
			case video.ElementPlayer0:
				fallthrough
			case video.ElementMissile0:
				ff.win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUP0], px, ff.to, 0xfe, func() { ff.resolve(offsetIdx + 1) })
			case video.ElementPlayer1:
				fallthrough
			case video.ElementMissile1:
				ff.win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUP1], px, ff.to, 0xfe, func() { ff.resolve(offsetIdx + 1) })
			}
		})
	})
}
