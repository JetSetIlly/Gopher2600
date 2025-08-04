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

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox/engines"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/imgui-go/v5"
)

type playscrSubtitles struct {
	img       *SdlImgui
	subtitles strings.Builder
}

func (psub *playscrSubtitles) draw() {
	w, h := psub.img.plt.windowSize()
	h *= 0.85

	// only show the most recent subtitle 'sentence'
	sub := strings.TrimSpace(psub.subtitles.String())
	splt := strings.Split(sub, engines.SubtitleSentence)
	if len(splt) > 1 && len(splt[len(splt)-1]) > 0 {
		sub = splt[len(splt)-1]
	}
	sub = strings.TrimSpace(sub)
	if len(sub) == 0 {
		return
	}

	imgui.PushStyleColor(imgui.StyleColorWindowBg, psub.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorBorder, psub.img.cols.Transparent)
	defer imgui.PopStyleColorV(2)

	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.Vec2{})
	defer imgui.PopStyleVarV(1)

	p := imgui.Vec2{X: 0.0, Y: h}
	imgui.SetNextWindowPos(p)
	imgui.SetNextWindowSize(imgui.Vec2{X: w, Y: h})

	imgui.BeginV("##subtitles", nil, imgui.WindowFlagsAlwaysAutoResize|
		imgui.WindowFlagsNoScrollbar|imgui.WindowFlagsNoTitleBar|
		imgui.WindowFlagsNoDecoration|imgui.WindowFlagsNoSavedSettings|
		imgui.WindowFlagsNoBringToFrontOnFocus)
	defer imgui.End()

	imgui.PushFont(psub.img.fonts.subtitles[psub.img.fonts.subtitlesIdx])
	defer imgui.PopFont()

	padding := float32(5.0)

	sz := imgui.CalcTextSize(sub, false, 0.0)
	p = imgui.CursorScreenPos()
	p.X = (w - sz.X) / 2
	imgui.SetCursorScreenPos(p)

	p = p.Plus(imgui.Vec2{X: -padding, Y: padding / 2})
	dl := imgui.WindowDrawList()
	dl.AddRectFilled(p, p.Plus(sz).Plus(imgui.Vec2{X: padding * 2, Y: -padding}), psub.img.cols.subtitlesBackground)

	imgui.PushStyleColor(imgui.StyleColorText, psub.img.cols.SubtitlesText)
	imgui.Text(sub)
	imgui.PopStyleColor()
}

func (psub *playscrSubtitles) set(v any, args ...any) {
	switch n := v.(type) {
	case notifications.Notice:
		switch n {
		case notifications.NotifyAtariVoxSubtitle:
			s := args[0].([]gui.FeatureReqData)[0]
			if s == engines.StaleSubtitle {
				psub.subtitles.Reset()
			} else {
				psub.subtitles.WriteString(s.(string))
			}
		}
	}
}
