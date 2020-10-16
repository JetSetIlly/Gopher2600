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
	"github.com/inkyblackness/imgui-go/v2"
)

const winAudioTitle = "Audio"

type winAudio struct {
	windowManagement
	img *SdlImgui

	displayBuffer []float32
	newData       chan float32
}

func newWinAudio(img *SdlImgui) (managedWindow, error) {
	win := &winAudio{
		img:           img,
		displayBuffer: make([]float32, 2048),
		newData:       make(chan float32, 2048),
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

	imgui.SetNextWindowPosV(imgui.Vec2{648, 440}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(winAudioTitle, &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.AudioOscBg)
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.AudioOscLine)
	imgui.PlotLines("", win.displayBuffer)
	imgui.PopStyleColor()
	imgui.PopStyleColor()
	imgui.End()

	done := false
	ct := 0
	for !done {
		select {
		case d := <-win.newData:
			ct++
			win.displayBuffer = append(win.displayBuffer, d)
		default:
			done = true
			win.displayBuffer = win.displayBuffer[ct:]
		}
	}
}

// SetAudio implements television.AudioMixer.
func (win *winAudio) SetAudio(audioData uint8) error {
	select {
	case win.newData <- float32(audioData) / 256:
	default:
	}
	return nil
}

// EndMixing implements television.AudioMixer.
func (win *winAudio) EndMixing() error {
	return nil
}
