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
	"fmt"
	"image"
	"image/color"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/cdf"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winCDFStreamsID = "CDF Streams"

type winCDFStreams struct {
	img  *SdlImgui
	open bool

	streamTextures [cdf.NumDatastreams]uint32
	streamPixels   [cdf.NumDatastreams]*image.RGBA

	imageSize image.Point

	trackScreen bool
	scanlines   int32

	popupPalette *popupPalette
	streamColour uint8
	background   uint8

	scaling float32
}

func newWinCDFStreams(img *SdlImgui) (window, error) {
	win := &winCDFStreams{
		img:          img,
		scanlines:    specification.AbsoluteMaxScanlines,
		trackScreen:  true,
		popupPalette: newPopupPalette(img),
		streamColour: 04,
		background:   00,
		scaling:      1.25,
	}

	for i := range win.streamTextures {
		gl.GenTextures(1, &win.streamTextures[i])
		gl.BindTexture(gl.TEXTURE_2D, win.streamTextures[i])
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	}

	for i := range win.streamTextures {
		win.streamPixels[i] = image.NewRGBA(image.Rect(0, 0, 8, specification.AbsoluteMaxScanlines))
		imageSize := win.streamPixels[i].Bounds().Size()

		gl.BindTexture(gl.TEXTURE_2D, win.streamTextures[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(imageSize.X), int32(imageSize.Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(win.streamPixels[i].Pix))
	}

	win.imageSize = win.streamPixels[0].Bounds().Size()
	for y := 0; y < win.imageSize.Y; y++ {
		for x := 0; x < win.imageSize.X; x++ {
			for i := range win.streamPixels {
				win.streamPixels[i].SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
			}
		}
	}

	win.refreshTextures()

	return win, nil
}

func (win *winCDFStreams) init() {
}

func (win *winCDFStreams) id() string {
	return winCDFStreamsID
}

func (win *winCDFStreams) isOpen() bool {
	return win.open
}

func (win *winCDFStreams) setOpen(open bool) {
	win.open = open
}

func (win *winCDFStreams) updateStreams() {
	// do not open window if there is no valid cartridge debug bus available
	r, ok := win.img.lz.Cart.Registers.(cdf.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	mem := win.img.lz.Cart.Static

	// keep track of scanlines
	scanlines := win.img.lz.TV.FrameInfo.VisibleBottom - win.img.lz.TV.FrameInfo.VisibleTop
	if !win.trackScreen {
		scanlines = int(win.scanlines)
	} else {
		win.scanlines = int32(scanlines)
	}

	_, _, pal := win.img.imguiTVPalette()
	col := pal[win.streamColour]
	bg := pal[win.background]
	col = col.Times(255)
	bg = bg.Times(255)

	// draw pixels
	for i := range r.Datastream {
		for y := 0; y < win.imageSize.Y; y++ {
			v := r.Datastream[i].Peek(y, mem)

			for x := 0; x < 8; x++ {
				if y <= scanlines {
					if (v<<x)&0x80 == 0x80 {
						win.streamPixels[i].SetRGBA(x, y, color.RGBA{R: uint8(col.X), G: uint8(col.Y), B: uint8(col.Z), A: 255})
					} else {
						win.streamPixels[i].SetRGBA(x, y, color.RGBA{R: uint8(bg.X), G: uint8(bg.Y), B: uint8(bg.Z), A: 255})
					}
				} else {
					win.streamPixels[i].SetRGBA(x, y, color.RGBA{R: uint8(bg.X), G: uint8(bg.Y), B: uint8(bg.Z), A: 100})
				}
			}
		}
	}

	win.refreshTextures()
}

func (win *winCDFStreams) refreshTextures() {
	for i := range win.streamTextures {
		gl.BindTexture(gl.TEXTURE_2D, win.streamTextures[i])
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(win.imageSize.X), int32(win.imageSize.Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(win.streamPixels[i].Pix))
	}
}

func (win *winCDFStreams) draw() {
	if !win.open {
		return
	}

	if !win.img.lz.Cart.HasStaticBus {
		return
	}

	// do not open window if there is no valid cartridge debug bus available
	r, ok := win.img.lz.Cart.Registers.(cdf.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return
	}

	imgui.SetNextWindowPosV(imgui.Vec2{100, 100}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)

	win.updateStreams()

	for i := 0; i < len(win.streamTextures); i++ {
		imgui.BeginGroup()

		imgui.Image(imgui.TextureID(win.streamTextures[i]), imgui.Vec2{
			X: float32(win.imageSize.X) * (win.scaling + 1),
			Y: float32(win.imageSize.Y) * win.scaling,
		})

		imguiTooltip(func() {
			imgui.Text("Datastream ")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmLocation)
			imgui.Text(fmt.Sprintf("%d", i))
			imgui.PopStyleColor()
			imgui.Separator()

			imgui.Text("Pointer:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
			imgui.Text(fmt.Sprintf("%08x", r.Datastream[i].AfterCALLFN))
			imgui.PopStyleColor()

			imgui.Text("Increment:")
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmAddress)
			imgui.Text(fmt.Sprintf("%08x", r.Datastream[i].Increment))
			imgui.PopStyleColor()
		}, true)

		imgui.EndGroup()
		imgui.SameLine()
	}

	imgui.Spacing()
	imgui.Spacing()

	imguiLabel("Stream length")
	if win.trackScreen {
		imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
		imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
	}
	imgui.SliderInt("##streamlength", &win.scanlines, 100, specification.AbsoluteMaxScanlines)
	if win.trackScreen {
		imgui.PopItemFlag()
		imgui.PopStyleVar()
	}

	imgui.SameLineV(0, 20)
	imgui.Checkbox("Track Screen Size", &win.trackScreen)

	imgui.SameLineV(0, 40)
	if win.img.imguiSwatch(win.streamColour, 0.75) {
		win.popupPalette.request(&win.streamColour, win.updateStreams)
	}
	imgui.AlignTextToFramePadding()
	imgui.Text("Colour")

	imgui.SameLineV(0, 20)
	if win.img.imguiSwatch(win.background, 0.75) {
		win.popupPalette.request(&win.background, win.updateStreams)
	}
	imgui.AlignTextToFramePadding()
	imgui.Text("Background")

	win.popupPalette.draw()

	imgui.End()
}

func (win *winCDFStreams) isStreamTexture(id uint32) bool {
	if !win.open {
		return false
	}

	for _, i := range win.streamTextures {
		if id == i {
			return true
		}
	}

	return false
}
