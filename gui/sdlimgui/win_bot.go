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
	"crypto/sha1"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/bots"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winBotID = "Bot"

type winBot struct {
	playmodeWin

	img *SdlImgui

	// the location at which the preview image is being drawn
	screenOrigin imgui.Vec2

	// for now scaling is very simple when compared to dbgscr
	screenScalingX float32
	screenScalingY float32

	obsImage     *image.RGBA
	obsTexture   texture
	mouseTexture texture

	selectionStart imgui.Vec2
	selectionEnd   imgui.Vec2
	selectActive   bool
	selection      image.Rectangle

	diagnostics []bots.Diagnostic
	dirty       bool

	// render channels are given to use by the main emulation through a GUI request
	feedback *bots.Feedback
}

// width of bot image never changes
const botImageScaling = 1.0
const botImageWidth = botImageScaling * specification.ClksScanline * pixelWidth
const botImageHeight = botImageScaling * specification.AbsoluteMaxScanlines

func newWinBot(img *SdlImgui) (window, error) {
	win := &winBot{
		img:            img,
		diagnostics:    make([]bots.Diagnostic, 0, 1024),
		screenScalingX: botImageScaling * pixelWidth,
		screenScalingY: botImageScaling,
	}

	win.obsTexture = img.rnd.addTexture(shaderColor, true, true, nil)
	win.mouseTexture = img.rnd.addTexture(shaderColor, true, true, nil)

	return win, nil
}

func (win *winBot) init() {
}

func (win winBot) id() string {
	return winBotID
}

// start bot session will effectively end a bot session if feedback channels are nil
func (win *winBot) startBotSession(feedback *bots.Feedback) {
	win.feedback = feedback
	win.obsTexture.clear()
	win.mouseTexture.clear()
	win.diagnostics = win.diagnostics[:]
}

func (win *winBot) playmodeDraw() bool {
	if win.feedback == nil {
		win.playmodeWin.playmodeSetOpen(false)
		return false
	}

	// receive new thumbnail data and copy to texture
	select {
	case win.obsImage = <-win.feedback.Images:
		if win.obsImage != nil {
			win.obsTexture.markForCreation()
			win.obsTexture.render(win.obsImage)
		}
	default:
	}

	done := false
	for !done {
		select {
		case d := <-win.feedback.Diagnostic:
			if len(win.diagnostics) == cap(win.diagnostics) {
				win.diagnostics = win.diagnostics[1:]
			}
			win.diagnostics = append(win.diagnostics, d)
			win.dirty = true
		default:
			done = true
		}
	}

	if !win.playmodeOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 75, Y: 75}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 500, Y: 525}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.playmodeGeom.update()
	imgui.End()

	return true
}

func (win *winBot) draw() {
	// add padding around screen image
	padding := imgui.Vec2{X: (imgui.ContentRegionAvail().X - botImageWidth) / 2, Y: 0}
	imgui.SetCursorPos(imgui.CursorPos().Plus(padding))

	win.screenOrigin = imgui.CursorScreenPos()
	imgui.Image(imgui.TextureID(win.obsTexture.getID()), imgui.Vec2{X: botImageWidth, Y: botImageHeight})
	imgui.SetCursorScreenPos(win.screenOrigin)
	win.drawMouseLayer()

	if imgui.BeginChildV("##log", imgui.Vec2{}, true, imgui.WindowFlagsAlwaysAutoResize) {
		var clipper imgui.ListClipper
		clipper.Begin(len(win.diagnostics))
		for clipper.Step() {
			for i := clipper.DisplayStart; i < clipper.DisplayEnd; i++ {
				imgui.Text(win.diagnostics[i].Group)
				for _, s := range strings.Split(win.diagnostics[i].Diagnostic, "\n") {
					imgui.SameLine()
					imgui.Text(s)
				}
			}
		}

		if win.dirty {
			imgui.SetScrollHereY(0.0)
			win.dirty = false
		}
		imgui.EndChild()
	}
}

func (win *winBot) drawMouseLayer() {
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.ImageButton("bot_mouse_layer", imgui.TextureID(win.mouseTexture.getID()), imgui.Vec2{X: botImageWidth, Y: botImageHeight})
	imgui.PopStyleColorV(3)

	if imgui.IsWindowFocused() && imgui.IsItemHovered() {
		if imgui.IsMouseDown(0) {
			if !win.selectActive {
				win.selectionStart = imgui.MousePos().Minus(win.screenOrigin)
				win.selectActive = true
			} else {
				win.selectionEnd = imgui.MousePos().Minus(win.screenOrigin)

				minX := int(win.selectionStart.X / win.screenScalingX)
				minY := int(win.selectionStart.Y / win.screenScalingY)
				maxX := int(win.selectionEnd.X / win.screenScalingX)
				maxY := int(win.selectionEnd.Y / win.screenScalingY)
				win.selection = image.Rect(minX, minY, maxX, maxY)

				mouseImage := image.NewRGBA(image.Rect(0, 0, specification.ClksScanline, specification.AbsoluteMaxScanlines))
				col := color.RGBA{200, 50, 50, 100}
				draw.Draw(mouseImage, win.selection, &image.Uniform{col}, image.Point{}, draw.Over)

				win.mouseTexture.markForCreation()
				win.mouseTexture.render(mouseImage)
			}
		} else if win.selectActive {
			win.selectActive = false

			selectionImage := image.NewRGBA(win.selection)
			draw.Draw(selectionImage, win.selection, win.obsImage.SubImage(win.selection), win.selection.Min, draw.Src)
			fmt.Printf("%#v\n", win.selection)
			fmt.Printf("%#v\n", sha1.Sum(selectionImage.Pix))
		}
	}
}
