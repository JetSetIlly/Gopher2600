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
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/cdf"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
)

const winCDFStreamsID = "CDF Streams"

// keeps track of whether the datastream drag and drop is active and which
// datastreams are involved (ie. source and target)
type datastreamDragAndDrop struct {
	active bool
	src    int
	tgt    int

	previousSrc int
}

type winCDFStreams struct {
	debuggerWin

	img *SdlImgui

	streamPixels   [cdf.NumDatastreams]*image.RGBA
	streamTextures [cdf.NumDatastreams]uint32
	pixelsSize     image.Point

	colouriser   datastreamDragAndDrop
	colourSource [cdf.NumDatastreams]int

	trackScreen bool
	scanlines   int32

	scaling float32

	detailTexture uint32

	optionsHeight float32
}

func newWinCDFStreams(img *SdlImgui) (window, error) {
	win := &winCDFStreams{
		img:         img,
		scanlines:   specification.AbsoluteMaxScanlines,
		trackScreen: true,
		scaling:     1.49,
	}

	win.clearColourStreaming()

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

	win.pixelsSize = win.streamPixels[0].Bounds().Size()
	for y := 0; y < win.pixelsSize.Y; y++ {
		for x := 0; x < win.pixelsSize.X; x++ {
			for i := range win.streamPixels {
				win.streamPixels[i].SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 0, A: 255})
			}
		}
	}

	win.refreshTextures()

	// tooltip detail
	gl.GenTextures(1, &win.detailTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.detailTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

func (win *winCDFStreams) init() {
}

func (win *winCDFStreams) id() string {
	return winCDFStreamsID
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

	fg := color.RGBA{100, 100, 100, 255}
	bg := color.RGBA{10, 10, 10, 255}
	unused := color.RGBA{10, 10, 10, 100}

	_, _, _, pal := win.img.imguiTVPalette()

	// draw pixels
	for i := range r.Datastream {
		for y := 0; y < win.pixelsSize.Y; y++ {
			// pixel data
			v := r.Datastream[i].Peek(y, mem)

			// colour source
			col := fg
			if win.colouriser.active && win.colouriser.tgt == i {
				s := r.Datastream[win.colouriser.src].Peek(y, mem)
				col = pal[s]
			} else if win.colourSource[i] > -1 {
				s := r.Datastream[win.colourSource[i]].Peek(y, mem)
				col = pal[s]
			}

			// plot pixels
			for x := 0; x < 8; x++ {
				if y <= scanlines {
					if (v<<x)&0x80 == 0x80 {
						win.streamPixels[i].SetRGBA(x, y, col)
					} else {
						win.streamPixels[i].SetRGBA(x, y, bg)
					}
				} else {
					win.streamPixels[i].SetRGBA(x, y, unused)
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
			0, 0, int32(win.pixelsSize.X), int32(win.pixelsSize.Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(win.streamPixels[i].Pix))
	}
}

func (win *winCDFStreams) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	if !win.img.lz.Cart.HasStaticBus {
		return false
	}

	// do not open window if there is no valid cartridge debug bus available
	_, ok := win.img.lz.Cart.Registers.(cdf.Registers)
	if !win.img.lz.Cart.HasRegistersBus || !ok {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{100, 100}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{920, 554}, imgui.ConditionFirstUseEver)
	imgui.SetNextWindowSizeConstraints(imgui.Vec2{551, 300}, imgui.Vec2{920, 554})

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsHorizontalScrollbar) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winCDFStreams) draw() {
	r := win.img.lz.Cart.Registers.(cdf.Registers)
	mem := win.img.lz.Cart.Static

	if imgui.BeginChildV("##stream", imgui.Vec2{Y: imguiRemainingWinHeight() - win.optionsHeight}, false, imgui.WindowFlagsNone) {
		win.updateStreams()

		// disable preview color. it will be turned on if drag and drop is being used this frame.
		win.colouriser.active = false

		for i := 0; i < len(win.streamTextures); i++ {
			imgui.BeginGroup()

			// styling for datastream buttons )including the image button)
			imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
			imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
			imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
			imgui.PushStyleColor(imgui.StyleColorDragDropTarget, win.img.cols.Transparent)
			imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{})

			// using button for labelling
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DataStreamNumLabel)
			imgui.ButtonV(fmt.Sprintf("%02d", i), imgui.Vec2{X: float32(win.pixelsSize.X) * (win.scaling + 1)})
			imgui.PopStyleColor()

			// position of ImageButton() we'll use this to measure the position of
			// the mouse in relation to the top/left of the button
			pos := imgui.CursorScreenPos()

			// Have to use ImageButton() rather than Image() because we want to use
			// drag and drop
			imgui.ImageButton(imgui.TextureID(win.streamTextures[i]), imgui.Vec2{
				X: float32(win.pixelsSize.X) * (win.scaling + 1),
				Y: float32(win.pixelsSize.Y) * win.scaling,
			})

			if imgui.IsItemHovered() {
				// quickly repeat previous drag and drop with a double click
				if imgui.IsMouseDoubleClicked(0) {
					win.colourSource[i] = win.colouriser.previousSrc
					win.updateStreams()
				}

				// clear assignment of color datastream
				if imgui.IsMouseClicked(1) {
					win.colourSource[i] = -1
					win.updateStreams()
				}
			}

			// the name of the drag and drop rendezvous
			const dragDropName = "DATASTREAM"

			// data stream image can be dragged. the drag image is a paintbrush
			imgui.PushStyleVarFloat(imgui.StyleVarPopupBorderSize, 0.0)
			imgui.PushStyleColor(imgui.StyleColorPopupBg, win.img.cols.Transparent)
			if imgui.BeginDragDropSource(imgui.DragDropFlagsNone) {
				imgui.SetDragDropPayload(dragDropName, []byte{byte(i)}, imgui.ConditionAlways)
				imgui.PushFont(win.img.glsl.fonts.largeFontAwesome)
				imgui.Text(string(fonts.PaintBrush))
				imgui.PopFont()
				imgui.EndDragDropSource()

				// drag and drop is active
				win.colouriser.active = true
				win.colouriser.src = i
			}
			imgui.PopStyleColor()
			imgui.PopStyleVar()

			// each datastream image can also be dropped onto
			if imgui.BeginDragDropTarget() {
				// drag and drop is hovering over a legitimate target
				payload := imgui.AcceptDragDropPayload(dragDropName, imgui.DragDropFlagsAcceptPeekOnly)
				if payload != nil {
					// drag and drop is active. note that we may see the drop
					// target before we see the drop source, so setting active here
					// is required
					win.colouriser.active = true
					win.colouriser.tgt = i
					win.updateStreams()
				}

				// drag and drop has ended on a legitimate target
				payload = imgui.AcceptDragDropPayload(dragDropName, imgui.DragDropFlagsNone)
				if payload != nil {
					win.colourSource[i] = int(payload[0])
					win.colouriser.previousSrc = win.colouriser.src
					win.updateStreams()
				}
				imgui.EndDragDropTarget()
			}

			imgui.PopStyleVar()
			imgui.PopStyleColorV(4)

			win.img.imguiTooltip(func() {
				imgui.Text("Datastream ")
				imgui.SameLine()
				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmLocation)
				imgui.Text(fmt.Sprintf("%d", i))
				imgui.PopStyleColor()

				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()

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

				// mouse position is used to decide which values in the stream
				// to peek/show
				p := imgui.MousePos()
				p = p.Minus(pos)

				const numOfAdditionalPeeks = 3

				y := int(p.Y / win.scaling)
				yTop := y - numOfAdditionalPeeks
				if yTop < 0 {
					yTop = 0
				}
				yBot := y + numOfAdditionalPeeks
				if yBot >= int(win.scanlines) {
					yBot = int(win.scanlines)
				}

				// test if mouse position intersects with the active part of
				// the texture
				if y >= 0 && y < int(win.scanlines) {
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()

					// list of values
					imgui.BeginGroup()
					imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.5)
					for yy := yTop; yy < y; yy++ {
						v := r.Datastream[i].Peek(yy, mem)
						imgui.Text(fmt.Sprintf("%03d %c %02x", yy, fonts.CaretRight, v))
					}
					imgui.PopStyleVar()

					v := r.Datastream[i].Peek(y, mem)
					imgui.Text(fmt.Sprintf("%03d %c %02x", y, fonts.CaretRight, v))

					imgui.PushStyleVarFloat(imgui.StyleVarAlpha, 0.5)
					for yy := y + 1; yy <= yBot; yy++ {
						v := r.Datastream[i].Peek(yy, mem)
						imgui.Text(fmt.Sprintf("%03d %c %02x", yy, fonts.CaretRight, v))
					}
					imgui.PopStyleVar()
					imgui.EndGroup()

					// detail texture
					imgui.SameLineV(0, 20)

					// small offset to help center the detail with the "list of
					// values" above
					p := imgui.CursorScreenPos()
					p.Y += imgui.CurrentStyle().FramePadding().Y
					imgui.SetCursorScreenPos(p)

					imgui.BeginGroup()

					// crop the pixels from the underlying stream texture
					detailCrop := image.Rect(0, y-numOfAdditionalPeeks, win.pixelsSize.X, y+numOfAdditionalPeeks+1)
					detailPixels := win.streamPixels[i].SubImage(detailCrop).(*image.RGBA)
					sz := detailPixels.Bounds().Size()

					gl.BindTexture(gl.TEXTURE_2D, win.detailTexture)
					gl.TexImage2D(gl.TEXTURE_2D, 0,
						gl.RGBA, int32(sz.X), int32(sz.Y), 0,
						gl.RGBA, gl.UNSIGNED_BYTE,
						gl.Ptr(detailPixels.Pix))

					// height of image matches the height of the "list of
					// values" above
					h := imgui.FontSize() + (imgui.CurrentStyle().FramePadding().Y)
					imgui.Image(imgui.TextureID(win.detailTexture), imgui.Vec2{
						X: float32(sz.X) * h * 1.25,
						Y: float32(sz.Y) * h,
					})

					imgui.EndGroup()
				}

				if win.colourSource[i] != -1 {
					imgui.Spacing()
					imgui.Separator()
					imgui.Spacing()
					imgui.Text(fmt.Sprintf("%c from datastream %d", fonts.PaintBrush, win.colourSource[i]))
				}

			}, true)

			imgui.EndGroup()
			imgui.SameLine()
		}

		imgui.EndChild()
	}

	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Spacing()

		imguiLabel("Stream length")
		if win.trackScreen {
			imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		}
		imgui.PushItemWidth(200)
		imgui.SliderInt("##streamlength", &win.scanlines, 100, specification.AbsoluteMaxScanlines)
		imgui.PopItemWidth()
		if win.trackScreen {
			imgui.PopItemFlag()
			imgui.PopStyleVar()
		}

		imgui.SameLineV(0, 20)
		imgui.Checkbox("Track Screen Size", &win.trackScreen)

		// clear colours button is sometimes disabled
		imgui.SameLineV(0, 20)
		enableClearColours := false
		for _, v := range win.colourSource {
			if v != -1 {
				enableClearColours = true
				break
			}
		}
		if !enableClearColours {
			imgui.PushItemFlag(imgui.ItemFlagsDisabled, true)
			imgui.PushStyleVarFloat(imgui.StyleVarAlpha, disabledAlpha)
		}
		if imgui.Button("Clear Colours") {
			win.clearColourStreaming()
		}
		if !enableClearColours {
			imgui.PopStyleVar()
			imgui.PopItemFlag()
		}
	})
}

func (win *winCDFStreams) isStreamTexture(id uint32) bool {
	if !win.debuggerOpen {
		return false
	}

	if id == win.detailTexture {
		return true
	}

	for _, i := range win.streamTextures {
		if id == i {
			return true
		}
	}

	return false
}

func (win *winCDFStreams) clearColourStreaming() {
	for i := range win.colourSource {
		win.colourSource[i] = -1
	}
}
