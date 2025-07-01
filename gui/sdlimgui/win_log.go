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
	"strings"
	"time"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/imgui-go/v5"
)

const winLogID = "Log"

type winLog struct {
	debuggerWin
	img           *SdlImgui
	lastEntryTime time.Time
}

func newWinLog(img *SdlImgui) (window, error) {
	win := &winLog{
		img: img,
	}

	return win, nil
}

func (win *winLog) init() {
}

func (win *winLog) id() string {
	return winLogID
}

func (win *winLog) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 489, Y: 352}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 570, Y: 335}, imgui.ConditionFirstUseEver)

	imgui.PushStyleColor(imgui.StyleColorWindowBg, win.img.cols.LogBackground)
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}
	imgui.PopStyleColor()

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winLog) draw() {
	if imgui.BeginChild("##log") {
		logger.BorrowLog(func(log []logger.Entry) {
			imgui.ListClipperAll(len(log), func(i int) {
				imgui.Text(log[i].String())
			})

			// scroll to bottom if last entry in log is new
			if len(log) > 0 {
				lastEntry := log[len(log)-1]
				if lastEntry.Time != win.lastEntryTime {
					win.lastEntryTime = lastEntry.Time
					imgui.SetScrollHereY(0.0)
				}
			}
		})
	}
	imgui.EndChild()

	if imgui.BeginPopupContextItem() {
		if imgui.Selectable("Clear log") {
			logger.Clear()
		}
		if imgui.Selectable("Copy to clipboard") {
			win.copyToClipboard()
		}
		imgui.EndPopup()
	}

}

func (win *winLog) copyToClipboard() {
	logger.BorrowLog(func(log []logger.Entry) {
		var s strings.Builder
		for _, o := range log {
			s.WriteString(o.String())
			s.WriteString("\n")
		}
		win.img.plt.SetClipboardText(s.String())
	})
}
