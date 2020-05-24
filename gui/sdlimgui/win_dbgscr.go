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
	"fmt"
	"image"
	"strings"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v2"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/television"
)

const winDbgScrTitle = "TV Screen"

type winDbgScr struct {
	windowManagement
	widgetDimensions

	img *SdlImgui
	scr *screen

	// how to present the screen in the window
	pixelPerfect bool
	overlay      bool
	useAltPixels bool
	cropped      bool

	// textures
	screenTexture  uint32
	overlayTexture uint32

	// (re)create textures on next render()
	createTextures bool

	// is screen currently pointed at
	isHovered bool

	// the tv screen has captured mouse input
	isCaptured bool

	// is the popup break menu active
	isPopup bool

	// horizPos and scanline equivalent position of the mouse. only updated when isHovered is true
	horizPos int
	scanline int

	// height of tool bar at bottom of window. valid after first frame.
	toolBarHeight float32

	// additional padding for the image so that it is centered in its content space
	imagePadding imgui.Vec2

	// size of window and content area in which to center the image. we need
	// both depending on how we set the scaling from the screen. when resizing
	// the window, we use contentDim (the area inside the window) to figure out
	// the scaling value. when resizing numerically (with the getScale()
	// function) on the other hand, we scale the entire window accordingly
	winDim     imgui.Vec2
	contentDim imgui.Vec2

	// when set the scale value numerically (with the getScale() function) we
	// need to alter how we set the window size for the first frame afterwards.
	// the rescaled bool helps us do this.
	rescaled bool

	// the basic amount by which the image should be scaled. horizontal scaling
	// is slightly different (see horizScaling() function)
	scaling float32
}

func newWinDbgScr(img *SdlImgui) (managedWindow, error) {
	win := &winDbgScr{
		img:          img,
		scr:          img.screen,
		scaling:      2.0,
		pixelPerfect: true,
		cropped:      true,
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.overlayTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

func (win *winDbgScr) init() {
	win.widgetDimensions.init()
}

func (win *winDbgScr) destroy() {
}

func (win *winDbgScr) id() string {
	return winDbgScrTitle
}

func (win *winDbgScr) draw() {
	if !win.open {
		return
	}

	// actual display
	var w, h float32
	if win.cropped {
		w = win.getScaledWidth(true)
		h = win.getScaledHeight(true)
	} else {
		w = win.getScaledWidth(false)
		h = win.getScaledHeight(false)
	}

	imgui.SetNextWindowPosV(imgui.Vec2{8, 28}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})

	if win.rescaled {
		imgui.SetNextWindowSize(win.winDim)
		win.rescaled = false
	} else {
		imgui.SetNextWindowSizeV(imgui.Vec2{611, 470}, imgui.ConditionFirstUseEver)
	}

	// if isCaptured flag is set then change the title and border colors of the
	// TV Screen window.
	if win.isCaptured {
		imgui.PushStyleColor(imgui.StyleColorTitleBgActive, win.img.cols.CapturedScreenTitle)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.CapturedScreenBorder)
		defer imgui.PopStyleColorV(2)
	}

	// we don't want to ever show scrollbars
	imgui.BeginV(winDbgScrTitle, &win.open, imgui.WindowFlagsNoScrollbar)

	// note size of window and content area
	win.winDim = imgui.WindowSize()
	win.contentDim = imgui.ContentRegionAvail()

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// note the current cursor position. we'll use this to
	mouseOrigin := imgui.CursorScreenPos()

	// overlay texture on top of screen texture
	imgui.Image(imgui.TextureID(win.screenTexture), imgui.Vec2{w, h})
	if win.overlay {
		imgui.SetCursorScreenPos(mouseOrigin)
		imgui.Image(imgui.TextureID(win.overlayTexture), imgui.Vec2{w, h})
	}

	// [A] add the remaining horiz/vert padding around screen image [see B below]
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// popup menu on right mouse button
	// !TODO: RMB to release captured window causes popup to immediately open
	win.isPopup = imgui.BeginPopupContextItem()
	if win.isPopup {
		imgui.Text("Break")
		imgui.Separator()
		if imgui.Selectable(fmt.Sprintf("Scanline=%d", win.scanline)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d", win.scanline))
		}
		if imgui.Selectable(fmt.Sprintf("Horizpos=%d", win.horizPos)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK HP %d", win.horizPos))
		}
		if imgui.Selectable(fmt.Sprintf("Scanline=%d & Horizpos=%d", win.scanline, win.horizPos)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d & HP %d", win.scanline, win.horizPos))
		}
		imgui.EndPopup()
		win.isPopup = false
	}

	// if mouse is hovering over the image. note that if popup menu is active
	// then imgui.IsItemHovered() is false by definition
	win.isHovered = imgui.IsItemHovered()
	if win.isHovered {
		// *** CRIT SECTION
		win.scr.crit.section.RLock()

		// get mouse position and transform
		mp := imgui.MousePos().Minus(mouseOrigin)
		if win.cropped {
			sz := win.scr.crit.cropPixels.Bounds().Size()
			mp.X = mp.X / win.getScaledWidth(true) * float32(sz.X)
			mp.Y = mp.Y / win.getScaledHeight(true) * float32(sz.Y)
			mp.X += float32(television.HorizClksHBlank)
			mp.Y += float32(win.scr.crit.topScanline)
		} else {
			sz := win.scr.crit.pixels.Bounds().Size()
			mp.X = mp.X / win.getScaledWidth(false) * float32(sz.X)
			mp.Y = mp.Y / win.getScaledHeight(false) * float32(sz.Y)
		}

		win.horizPos = int(mp.X)
		win.scanline = int(mp.Y)

		// get reflection information
		var res reflection.ResultWithBank
		if win.horizPos < len(win.scr.crit.reflection) && win.scanline < len(win.scr.crit.reflection[win.horizPos]) {
			res = win.scr.crit.reflection[win.horizPos][win.scanline]
		}

		win.scr.crit.section.RUnlock()
		// *** CRIT SECTION END ***

		// present tooltip showing pixel coords and CPU state
		if !win.isCaptured {
			fmtRes, _ := win.img.lz.Dbg.Disasm.FormatResult(res.Bank, res.Res, disassembly.EntryLevelBlessed)
			if fmtRes.Address != "" {
				imgui.BeginTooltip()
				imgui.Text(fmt.Sprintf("Scanline: %d", win.scanline))
				imgui.Text(fmt.Sprintf("Horiz Pos: %d", win.horizPos-television.HorizClksHBlank))

				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
				if win.img.lz.Cart.NumBanks > 1 {
					imgui.Text(fmt.Sprintf("%s [bank %d]", fmtRes.Address, res.Bank))
				} else {
					imgui.Text(fmtRes.Address)
				}
				imgui.PopStyleColor()

				imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmMnemonic)
				imgui.Text(fmtRes.Mnemonic)
				imgui.PopStyleColor()

				if fmtRes.Operand != "" {
					imgui.SameLine()
					imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
					imgui.Text(fmtRes.Operand)
					imgui.PopStyleColor()
				}

				imgui.EndTooltip()
			}
		}
	}

	// start of tool bar
	toolBarTop := imgui.CursorPosY()

	// [B] we put spacing here otherwise the [A] leaves the cursor in the wrong position
	imgui.Spacing()

	// tv status line
	imguiText("Frame:")
	imguiText(fmt.Sprintf("%-4d", win.img.lz.TV.Frame))
	imgui.SameLineV(0, 15)
	imguiText("Scanline:")
	imguiText(fmt.Sprintf("%-4d", win.img.lz.TV.Scanline))
	imgui.SameLineV(0, 15)
	imguiText("Horiz Pos:")
	imguiText(fmt.Sprintf("%-4d", win.img.lz.TV.HP))

	// fps indicator
	imgui.SameLineV(0, 20)
	imgui.AlignTextToFramePadding()
	if win.img.paused {
		imguiText("no fps")
	} else {
		if win.img.lz.TV.ReqFPS < 1.0 {
			imguiText("< 1 fps")
		} else {
			imguiText(fmt.Sprintf("%03.1f fps", win.img.lz.TV.AcutalFPS))
		}
	}

	// include tv signal information
	imgui.SameLineV(0, 20)
	signal := strings.Builder{}
	if win.img.lz.TV.LastSignal.VSync {
		signal.WriteString("VSYNC ")
	}
	if win.img.lz.TV.LastSignal.VBlank {
		signal.WriteString("VBLANK ")
	}
	if win.img.lz.TV.LastSignal.CBurst {
		signal.WriteString("CBURST ")
	}
	if win.img.lz.TV.LastSignal.HSync {
		signal.WriteString("HSYNC ")
	}
	imgui.Text(signal.String())

	// display toggles
	imgui.Spacing()
	imgui.Checkbox("Debug Colours", &win.useAltPixels)
	imgui.SameLine()
	if imgui.Checkbox("Cropping", &win.cropped) {
		win.setCropping(win.cropped)
	}
	imgui.SameLine()
	imgui.Checkbox("Pixel Perfect", &win.pixelPerfect)
	imgui.SameLine()
	imgui.Checkbox("Overlay", &win.overlay)

	// note height of tool bar
	win.toolBarHeight = imgui.CursorPosY() - toolBarTop

	imgui.End()
}

func (win *winDbgScr) setOverlay(set bool) {
	win.overlay = set
}

func (win *winDbgScr) setCropping(set bool) {
	win.cropped = set
	win.createTextures = true
}

func (win *winDbgScr) resize() {
	win.createTextures = true
}

// render is called by service loop
func (win *winDbgScr) render() {
	win.scr.crit.section.RLock()
	defer win.scr.crit.section.RUnlock()

	var pixels *image.RGBA
	var overlayPixels *image.RGBA

	if win.cropped {
		if win.useAltPixels {
			pixels = win.scr.crit.cropAltPixels
		} else {
			pixels = win.scr.crit.cropPixels
		}
		overlayPixels = win.scr.crit.cropRefPixels
	} else {
		if win.useAltPixels {
			pixels = win.scr.crit.altPixels
		} else {
			pixels = win.scr.crit.pixels
		}
		overlayPixels = win.scr.crit.overlayPixels
	}

	gl.PixelStorei(gl.UNPACK_ROW_LENGTH, int32(pixels.Stride)/4)
	defer gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)

	gl.ActiveTexture(gl.TEXTURE0)

	if win.createTextures {
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(overlayPixels.Pix))

		win.createTextures = false

	} else {
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(overlayPixels.Pix))
	}

	// set screen image scaling (and image padding) based on the current window size
	win.setScaleFromWindow(win.contentDim)
}

func (win *winDbgScr) getScaledWidth(cropped bool) float32 {
	if cropped {
		return float32(win.scr.crit.cropPixels.Bounds().Size().X) * win.getScaling(true)
	}
	return float32(win.scr.crit.pixels.Bounds().Size().X) * win.getScaling(true)
}

func (win *winDbgScr) getScaledHeight(cropped bool) float32 {
	if cropped {
		return float32(win.scr.crit.cropPixels.Bounds().Size().Y) * win.getScaling(false)
	}
	return float32(win.scr.crit.pixels.Bounds().Size().Y) * win.getScaling(false)
}

func (win *winDbgScr) setScaleFromWindow(sz imgui.Vec2) {
	// must be called from with a critical section

	sz.Y -= win.toolBarHeight
	winAspectRatio := sz.X / sz.Y

	var imageW float32
	var imageH float32
	if win.cropped {
		imageW = float32(win.scr.crit.cropPixels.Bounds().Size().X)
		imageH = float32(win.scr.crit.cropPixels.Bounds().Size().Y)
	} else {
		imageW = float32(win.scr.crit.pixels.Bounds().Size().X)
		imageH = float32(win.scr.crit.pixels.Bounds().Size().Y)
	}
	imageW *= pixelWidth * win.scr.aspectBias

	aspectRatio := imageW / imageH

	if aspectRatio < winAspectRatio {
		win.scaling = sz.Y / imageH
		win.imagePadding = imgui.Vec2{X: float32(int((sz.X - (imageW * win.scaling)) / 2))}
	} else {
		win.scaling = sz.X / imageW
		win.imagePadding = imgui.Vec2{Y: float32(int((sz.Y - (imageH * win.scaling)) / 2))}
	}
}

func (win *winDbgScr) getScaling(horiz bool) float32 {
	if horiz {
		return float32(pixelWidth * win.scr.aspectBias * win.scaling)
	}
	return win.scaling
}

func (win *winDbgScr) setScaling(scaling float32) {
	win.rescaled = true
	win.winDim = win.winDim.Times(scaling / win.scaling)
}
