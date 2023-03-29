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

	"github.com/go-gl/gl/v2.1/gl"
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
	obsTexture   uint32
	mouseTexture uint32

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

	gl.GenTextures(1, &win.obsTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.GenTextures(1, &win.mouseTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.mouseTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

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

	gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, 1, 1, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr([]uint8{0}))

	gl.BindTexture(gl.TEXTURE_2D, win.mouseTexture)
	gl.TexImage2D(gl.TEXTURE_2D, 0,
		gl.RGBA, 1, 1, 0,
		gl.RGBA, gl.UNSIGNED_BYTE,
		gl.Ptr([]uint8{0}))

	win.diagnostics = win.diagnostics[:]
}

func (win *winBot) playmodeDraw() {
	// no bot feedback instance
	if win.feedback == nil {
		return
	}

	// receive new thumbnail data and copy to texture
	select {
	case win.obsImage = <-win.feedback.Images:
		if win.obsImage != nil {
			gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(win.obsImage.Stride)/4)
			defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

			gl.BindTexture(gl.TEXTURE_2D, win.obsTexture)
			gl.TexImage2D(gl.TEXTURE_2D, 0,
				gl.RGBA, int32(win.obsImage.Bounds().Size().X), int32(win.obsImage.Bounds().Size().Y), 0,
				gl.RGBA, gl.UNSIGNED_BYTE,
				gl.Ptr(win.obsImage.Pix))
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
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{75, 75}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{500, 525}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{500, 500}, imgui.Vec2{750, 800})

	if imgui.BeginV(win.playmodeID(win.id()), &win.playmodeOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.playmodeGeom.update()
	imgui.End()
}

func (win *winBot) draw() {
	// add padding around screen image
	padding := imgui.Vec2{X: (imgui.ContentRegionAvail().X - botImageWidth) / 2, Y: 0}
	imgui.SetCursorPos(imgui.CursorPos().Plus(padding))

	win.screenOrigin = imgui.CursorScreenPos()
	imgui.Image(imgui.TextureID(win.obsTexture), imgui.Vec2{botImageWidth, botImageHeight})
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
	imgui.ImageButton(imgui.TextureID(win.mouseTexture), imgui.Vec2{botImageWidth, botImageHeight})
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

				gl.BindTexture(gl.TEXTURE_2D, win.mouseTexture)
				gl.TexImage2D(gl.TEXTURE_2D, 0,
					gl.RGBA, int32(mouseImage.Bounds().Size().X), int32(mouseImage.Bounds().Size().Y), 0,
					gl.RGBA, gl.UNSIGNED_BYTE,
					gl.Ptr(mouseImage.Pix))
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
