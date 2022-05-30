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
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/logger"
)

func (win *winDbgScr) paintTarget() {
	// each datastream image can also be dropped onto
	if imgui.BeginDragDropTarget() {
		// drag and drop has ended on a legitimate target
		payload := imgui.AcceptDragDropPayload(painDragDrop, imgui.DragDropFlagsNone)
		if payload != nil {
			colour := payload[0]

			mouse := win.mouseCoords()
			if mouse.valid {
				resumeCoords := win.img.lz.TV.Coords
				resumeCoords.Scanline = mouse.scanline
				resumeCoords.Clock = mouse.clock
				paintCoords := resumeCoords
				paintCoords.Frame -= 1

				select {
				case win.noRenderSet <- true:
				default:
				}
				win.img.dbg.GotoCoords(paintCoords)

				win.img.dbg.PushRawEventImmediate(func() {
					ref := win.scr.crit.reflection[mouse.offset]
					logger.Logf("paint", "filling %s at %s", ref.VideoElement.String(), paintCoords)
					switch ref.VideoElement {
					case video.ElementBackground:
						px := uint8((ref.Signal & signal.Color) >> signal.ColorShift)
						win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUBK], px, colour, 0xfe)
					case video.ElementPlayfield:
						fallthrough
					case video.ElementBall:
						px := uint8((ref.Signal & signal.Color) >> signal.ColorShift)
						win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUPF], px, colour, 0xfe)
					case video.ElementPlayer0:
						fallthrough
					case video.ElementMissile0:
						px := uint8((ref.Signal & signal.Color) >> signal.ColorShift)
						win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUP0], px, colour, 0xfe)
					case video.ElementPlayer1:
						fallthrough
					case video.ElementMissile1:
						px := uint8((ref.Signal & signal.Color) >> signal.ColorShift)
						win.img.dbg.PushDeepPoke(cpubus.WriteAddress[cpubus.COLUP1], px, colour, 0xfe)
					}
				})

				win.img.dbg.PushRawEventImmediate(func() {
					win.img.dbg.GotoCoords(resumeCoords)
					select {
					case win.noRenderSet <- false:
					default:
					}
				})
			}
		}
		imgui.EndDragDropTarget()
	}
}
