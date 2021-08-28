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
	"math"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/gui/fonts"
)

const winAudioID = "Audio"

type winAudio struct {
	img  *SdlImgui
	open bool

	displayBuffer []float32
	newData       chan float32
	clearData     chan bool

	endabledDim imgui.Vec2
}

func newWinAudio(img *SdlImgui) (window, error) {
	win := &winAudio{
		img:       img,
		newData:   make(chan float32, 2048),
		clearData: make(chan bool, 1),
	}
	win.reset()

	img.tv.AddAudioMixer(win)

	return win, nil
}

func (win *winAudio) init() {
	imgui.PushFont(win.img.glsl.largeFontAwesome)
	defer imgui.PopFont()
	win.endabledDim = imguiGetFrameDim(string(fonts.AudioDisabled), string(fonts.AudioEnabled))
}

func (win *winAudio) id() string {
	return winAudioID
}

func (win *winAudio) isOpen() bool {
	return win.open
}

func (win *winAudio) setOpen(open bool) {
	win.open = open
}

func (win *winAudio) draw() {
	if !win.open {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{625, 567}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	imgui.PushStyleColor(imgui.StyleColorFrameBg, win.img.cols.AudioOscBg)
	imgui.PushStyleColor(imgui.StyleColorPlotLines, win.img.cols.AudioOscLine)
	imgui.PlotLinesV("", win.displayBuffer, 0, "", math.MaxFloat32, math.MaxFloat32, imgui.Vec2{Y: win.img.glsl.largeFontAwesomeSize * 1.2})
	imgui.PopStyleColor()
	imgui.PopStyleColor()
	imgui.SameLine()

	enabled := win.img.prefs.audioEnabled.Get().(bool)

	label := string(fonts.AudioDisabled)
	if enabled {
		label = string(fonts.AudioEnabled)
	}

	imgui.PushFont(win.img.glsl.largeFontAwesome)
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)

	if imgui.ButtonV(label, imgui.Vec2{X: win.endabledDim.X}) {
		win.img.prefs.audioEnabled.Set(!enabled)
	}

	imgui.PopStyleColorV(3)
	imgui.PopFont()

	imgui.End()

	done := false
	ct := 0
	for !done {
		select {
		case d := <-win.newData:
			ct++
			win.displayBuffer = append(win.displayBuffer, d)
		case <-win.clearData:
			win.reset()
		default:
			done = true
			win.displayBuffer = win.displayBuffer[ct:]
		}
	}
}

// SetAudio implements protocol.AudioMixer.
func (win *winAudio) SetAudio(audioData uint8) error {
	select {
	case win.newData <- float32(audioData) / 256:
	default:
	}
	return nil
}

// EndMixing implements protocol.AudioMixer.
func (win *winAudio) EndMixing() error {
	return nil
}

// Reset implements protocol.AudioMixer.
//
// Should not be called by the GUI gorountine. Use winAudio.reset() instead.
func (win *winAudio) Reset() {
	select {
	case win.clearData <- true:
	default:
		// if we don't succeed in not sending the clear signal it's not the end
		// of the world. the signal that has been queued will do the job for us
	}
}

func (win *winAudio) reset() {
	win.displayBuffer = make([]float32, 2048)
}
