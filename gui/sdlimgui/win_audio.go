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

const winAudioTitle = "Audio"

type winAudio struct {
	windowManagement
	img         *SdlImgui
	audioStream []float32
}

func newWinAudio(img *SdlImgui) (managedWindow, error) {
	win := &winAudio{
		img:         img,
		audioStream: make([]float32, 1, 2048),
	}

	img.tv.AddAudioMixer(win)

	return win, nil
}

func (win *winAudio) init() {
}

func (win *winAudio) destroy() {
}

func (win *winAudio) id() string {
	return winAudioTitle
}

func (win *winAudio) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{17, 677}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winAudioTitle, &win.open,
		imgui.WindowFlagsAlwaysAutoResize|imgui.WindowFlagsNoTitleBar)

	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.OscBg)
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.OscLine)
	imgui.PlotLines("", win.audioStream)
	imgui.PopStyleColor()
	imgui.PopStyleColor()
	imgui.End()

	win.audioStream = win.audioStream[:1]
}

func (win *winAudio) SetAudio(audioData uint8) error {
	win.audioStream = append(win.audioStream, (float32(audioData))/256)
	return nil
}

func (win *winAudio) FlushAudio() error {
	return nil
}

func (win *winAudio) PauseAudio(pause bool) error {
	return nil
}

func (win *winAudio) EndMixing() error {
	return nil
}
