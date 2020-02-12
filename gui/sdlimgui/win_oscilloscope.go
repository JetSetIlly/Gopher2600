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

package sdlimgui

import (
	"github.com/inkyblackness/imgui-go/v2"
)

const oscilloscopeTitle = "Oscilloscope"

type oscilloscope struct {
	img         *SdlImgui
	audioStream []float32
}

func newOscilloscope(img *SdlImgui) (*oscilloscope, error) {
	osc := &oscilloscope{
		img:         img,
		audioStream: make([]float32, 1, 2048),
	}

	img.tv.AddAudioMixer(osc)

	return osc, nil
}

// draw is called by service loop
func (osc *oscilloscope) draw() {
	if osc.img.vcs != nil {
		imgui.SetNextWindowPosV(imgui.Vec2{20, 681}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.BeginV(oscilloscopeTitle, nil,
			imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoTitleBar)

		imgui.PushStyleColor(imgui.StyleColorFrameBg, imgui.Vec4{0.21, 0.29, 0.23, 1.0})
		imgui.PushStyleColor(imgui.StyleColorPlotLines, imgui.Vec4{0.10, 0.97, 0.29, 1.0})
		imgui.PlotLines("", osc.audioStream)
		imgui.PopStyleColor()
		imgui.PopStyleColor()
		imgui.End()
	}

	osc.audioStream = osc.audioStream[:1]
}

func (osc *oscilloscope) SetAudio(audioData uint8) error {
	osc.audioStream = append(osc.audioStream, (float32(audioData))/256)
	return nil
}

func (osc *oscilloscope) FlushAudio() error {
	return nil
}

func (osc *oscilloscope) PauseAudio(pause bool) error {
	return nil
}

func (osc *oscilloscope) EndMixing() error {
	return nil
}
