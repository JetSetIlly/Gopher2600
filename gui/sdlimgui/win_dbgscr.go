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

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go/v3"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/reflection"
)

const winDbgScrID = "TV Screen"

type winDbgScr struct {
	img  *SdlImgui
	open bool

	// reference to screen data
	scr *screen

	// how to present the screen in the window
	debugColors bool
	cropped     bool
	crt         bool
	overlay     bool

	// textures
	screenTexture   uint32
	overlayTexture  uint32
	phosphorTexture uint32

	// (re)create textures on next render()
	createTextures bool

	// is screen currently pointed at
	isHovered bool

	// the tv screen has captured mouse input
	isCaptured bool

	// is the popup break menu active
	isPopup bool

	// horizPos and scanline equivalent position of the mouse. only updated when isHovered is true
	mouseHorizPos int
	mouseScanline int

	// height of tool bar at bottom of window. valid after first frame.
	toolbarHeight float32

	// additional padding for the image so that it is centred in its content space
	imagePadding imgui.Vec2

	// size of window and content area in which to centre the image. we need
	// both depending on how we set the scaling from the screen. when resizing
	// the window, we use contentDim (the area inside the window) to figure out
	// the scaling value. when resizing numerically (with the getScale()
	// function) on the other hand, we scale the entire window accordingly
	winDim          imgui.Vec2
	contentDim      imgui.Vec2
	specComboDim    imgui.Vec2
	overlayComboDim imgui.Vec2

	// when set the scale value numerically (with the getScale() function) we
	// need to alter how we set the window size for the first frame afterwards.
	// the rescaled bool helps us do this.
	rescaled bool

	// the basic amount by which the image should be scaled. horizontal scaling
	// is slightly different (see horizScaling() function)
	scaling float32
}

func newWinDbgScr(img *SdlImgui) (window, error) {
	win := &winDbgScr{
		img:     img,
		scr:     img.screen,
		scaling: 2.0,
		crt:     false,
		cropped: true,
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

	gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitDbgScr)
	gl.GenTextures(1, &win.phosphorTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

func (win *winDbgScr) init() {
	win.overlayComboDim = imguiGetFrameDim("", reflection.OverlayList...)
	win.specComboDim = imguiGetFrameDim("", specification.SpecList...)
}

func (win *winDbgScr) id() string {
	return winDbgScrID
}

func (win *winDbgScr) isOpen() bool {
	return win.open
}

func (win *winDbgScr) setOpen(open bool) {
	win.open = open
}

func (win *winDbgScr) draw() {
	if !win.open {
		return
	}

	win.scr.crit.section.Lock()
	defer win.scr.crit.section.Unlock()

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
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoScrollbar)

	// note size of remaining window and content area
	win.winDim = imgui.WindowSize()
	win.contentDim = imgui.ContentRegionAvail()

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// note the current cursor position. we'll use this to
	mouseOrigin := imgui.CursorScreenPos()

	// push style info for screen and overlay ImageButton(). we're using
	// ImageButton because an Image will not capture mouse events and pass them
	// to the parent window. this means that a click-drag on the screen/overlay
	// will move the window, which we don't want.
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{0.0, 0.0})

	// screen texture
	imgui.ImageButton(imgui.TextureID(win.screenTexture), imgui.Vec2{w, h})

	// overlay texture on top of screen texture
	if win.overlay {
		imgui.SetCursorScreenPos(mouseOrigin)
		imgui.ImageButton(imgui.TextureID(win.overlayTexture), imgui.Vec2{w, h})
	}

	// pop style info for screen and overlay textures
	imgui.PopStyleVar()
	imgui.PopStyleColorV(3)

	// [A] add the remaining horiz/vert padding around screen image [see B below]
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// popup menu on right mouse button
	//
	// we only call OpenPopup() if it's not already open. also, care taken to
	// avoid menu opening when releasing a captured mouse.
	if !win.isPopup && !win.isCaptured && imgui.IsItemHovered() && imgui.IsMouseDown(1) {
		imgui.OpenPopup("breakmenu")
	}

	if imgui.BeginPopup("breakmenu") {
		win.isPopup = true
		imgui.Text("Break")
		imguiSeparator()
		if imgui.Selectable(fmt.Sprintf("Scanline=%d", win.mouseScanline)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d", win.mouseScanline))
		}
		if imgui.Selectable(fmt.Sprintf("Horizpos=%d", win.mouseHorizPos)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK HP %d", win.mouseHorizPos))
		}
		if imgui.Selectable(fmt.Sprintf("Scanline=%d & Horizpos=%d", win.mouseScanline, win.mouseHorizPos)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d & HP %d", win.mouseScanline, win.mouseHorizPos))
		}
		imgui.EndPopup()
	} else {
		win.isPopup = false
	}

	// if mouse is hovering over the image. note that if popup menu is active
	// then imgui.IsItemHovered() is false by definition
	win.isHovered = imgui.IsItemHovered()

	// draw tool tip
	if win.isHovered {
		win.drawReflectionTooltip(mouseOrigin)

		// mouse click will cause the rewind goto coords to run only when the
		// emulation is paused
		if win.img.state == gui.StatePaused {
			if imgui.IsMouseReleased(0) {
				win.img.screen.gotoCoordsX = win.mouseHorizPos
				win.img.screen.gotoCoordsY = win.img.wm.dbgScr.mouseScanline
				win.img.lz.Dbg.PushGotoCoords(win.mouseScanline, win.mouseHorizPos-specification.HorizClksHBlank)
			}
		}
	}

	// start of tool bar
	win.toolbarHeight = measureHeight(func() {
		// [B] we put spacing here otherwise the [A] leaves the cursor in the wrong position
		imgui.Spacing()

		// tv status line
		imgui.PushItemWidth(win.specComboDim.X)
		if imgui.BeginComboV("##spec", win.img.lz.TV.Spec.ID, imgui.ComboFlagNoArrowButton) {
			for _, s := range specification.SpecList {
				if imgui.Selectable(s) {
					win.img.term.pushCommand(fmt.Sprintf("TV SPEC %s", s))
				}
			}
			imgui.EndCombo()
		}
		imgui.PopItemWidth()

		imgui.SameLineV(0, 15)
		imguiLabel("Frame:")
		imguiLabel(fmt.Sprintf("%-4d", win.img.lz.TV.Frame))
		imgui.SameLineV(0, 15)
		imguiLabel("Scanline:")
		imguiLabel(fmt.Sprintf("%-4d", win.img.lz.TV.Scanline))
		imgui.SameLineV(0, 15)
		imguiLabel("Horiz Pos:")
		imguiLabel(fmt.Sprintf("%-4d", win.img.lz.TV.HP))

		// fps indicator
		imgui.SameLineV(0, 20)
		imgui.AlignTextToFramePadding()
		if win.img.state != gui.StateRunning {
			imguiLabel("no fps")
		} else {
			if win.img.lz.TV.ReqFPS < 1.0 {
				imguiLabel("< 1 fps")
			} else {
				imguiLabel(fmt.Sprintf("%03.1f fps", win.img.lz.TV.AcutalFPS))
			}
		}

		// include tv signal information
		imgui.SameLineV(0, 20)
		imgui.Text(win.img.lz.TV.LastSignal.String())

		// display toggles
		imgui.Spacing()
		imgui.Checkbox("Debug Colours", &win.debugColors)
		imgui.SameLine()
		if imgui.Checkbox("Cropping", &win.cropped) {
			win.setCropping(win.cropped)
		}
		imgui.SameLine()
		imgui.Checkbox("CRT Effects", &win.crt)
		imgui.SameLine()
		imgui.Checkbox("Overlay", &win.overlay)
		imgui.SameLine()
		imgui.PushItemWidth(win.overlayComboDim.X)
		if imgui.BeginComboV("##overlay", win.img.screen.crit.overlay, imgui.ComboFlagNoArrowButton) {
			for _, s := range reflection.OverlayList {
				if imgui.Selectable(s) {
					win.img.screen.crit.overlay = s
					win.img.screen.replotOverlay()
				}
			}

			imgui.EndCombo()
		}
		imgui.PopItemWidth()

		// add capture information
		imgui.SameLine()
		c := imgui.CursorPos()
		c.X += 10
		if win.isCaptured {
			imgui.SetCursorPos(c)
			imgui.Text("RMB or ESC to release mouse")
		} else {
			imgui.SetCursorPos(c)
			if imgui.Button("Capture mouse") {
				win.img.setCapture(true)
			}
		}
	})

	imgui.End()
}

// called from within a win.scr.crit.section Lock().
func (win *winDbgScr) drawReflectionTooltip(mouseOrigin imgui.Vec2) {
	// get mouse position and transform
	mp := imgui.MousePos().Minus(mouseOrigin)
	if win.cropped {
		sz := win.scr.crit.cropPixels.Bounds().Size()
		mp.X = mp.X / win.getScaledWidth(true) * float32(sz.X)
		mp.Y = mp.Y / win.getScaledHeight(true) * float32(sz.Y)
		mp.X += float32(specification.HorizClksHBlank)
		mp.Y += float32(win.scr.crit.topScanline)
	} else {
		sz := win.scr.crit.pixels.Bounds().Size()
		mp.X = mp.X / win.getScaledWidth(false) * float32(sz.X)
		mp.Y = mp.Y / win.getScaledHeight(false) * float32(sz.Y)
	}

	win.mouseHorizPos = int(mp.X)
	win.mouseScanline = int(mp.Y)

	// get reflection information
	var ref reflection.VideoStep

	if win.mouseHorizPos < len(win.scr.crit.reflection) && win.mouseScanline < len(win.scr.crit.reflection[win.mouseHorizPos]) {
		ref = win.scr.crit.reflection[win.mouseHorizPos][win.mouseScanline]
	}

	// present tooltip showing pixel coords and CPU state
	if win.isCaptured {
		return
	}

	imgui.BeginTooltip()
	defer imgui.EndTooltip()

	imgui.Text(fmt.Sprintf("Scanline: %d", win.mouseScanline))
	imgui.Text(fmt.Sprintf("Horiz Pos: %d", win.mouseHorizPos-specification.HorizClksHBlank))

	if win.overlay {
		switch win.scr.crit.overlay {
		case "WSYNC":
		case "Collisions":
			imguiSeparator()

			imguiLabel("CXM0P ")
			drawCollision(win.img, ref.Collision.CXM0P, video.CollisionMask, func(_ uint8) {})
			imguiLabel("CXM1P ")
			drawCollision(win.img, ref.Collision.CXM1P, video.CollisionMask, func(_ uint8) {})
			imguiLabel("CXP0FB")
			drawCollision(win.img, ref.Collision.CXP0FB, video.CollisionMask, func(_ uint8) {})
			imguiLabel("CXP1FB")
			drawCollision(win.img, ref.Collision.CXP1FB, video.CollisionMask, func(_ uint8) {})
			imguiLabel("CXM0FB")
			drawCollision(win.img, ref.Collision.CXM0FB, video.CollisionMask, func(_ uint8) {})
			imguiLabel("CXM1FB")
			drawCollision(win.img, ref.Collision.CXM1FB, video.CollisionMask, func(_ uint8) {})
			imguiLabel("CXBLPF")
			drawCollision(win.img, ref.Collision.CXBLPF, video.CollisionCXBLPFMask, func(_ uint8) {})
			imguiLabel("CXPPMM")
			drawCollision(win.img, ref.Collision.CXPPMM, video.CollisionMask, func(_ uint8) {})

			imguiSeparator()

			s := ref.Collision.LastVideoCycle.String()
			if s != "" {
				imgui.Text(s)
			} else {
				imgui.Text("no new collision")
			}
		case "HMOVE":
			imguiSeparator()
			if ref.Hmove.Delay {
				imgui.Text(fmt.Sprintf("HMOVE delay: %d", ref.Hmove.DelayCt))
			} else if ref.Hmove.Latch {
				if ref.Hmove.RippleCt != 255 {
					imgui.Text(fmt.Sprintf("HMOVE ripple: %d", ref.Hmove.RippleCt))
				} else {
					imgui.Text("HMOVE latched")
				}
			} else {
				imgui.Text("no HMOVE")
			}
		case "Optimised":
			imguiSeparator()
			if ref.Optimisations.ReusePixel && ref.Optimisations.NoCollisionCheck {
				imgui.Text("Shortest render path used")
			} else {
				if !ref.Optimisations.ReusePixel {
					imgui.Text("Pixel col/layer recalculated")
				}
				if !ref.Optimisations.NoCollisionCheck {
					imgui.Text("Collision registers recalculated")
				}
			}
		}
		return
	}

	e, _ := win.img.lz.Dbg.Disasm.FormatResult(ref.Bank, ref.CPU, disassembly.EntryLevelBlessed)
	if e.Address == "" {
		return
	}

	imguiSeparator()

	// pixel swatch. using black swatch if pixel is HBLANKed or VBLANKed
	if ref.IsHblank || ref.TV.VBlank {
		win.img.imguiSwatch(0, 0.5)
	} else {
		win.img.imguiSwatch(uint8(ref.TV.Pixel), 0.5)
	}

	// element information regardless of HBLANK/VBLANK state
	imguiLabelEnd(ref.VideoElement.String())

	imgui.Spacing()

	// instruction information
	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
	if win.img.lz.Cart.NumBanks > 1 {
		imgui.Text(fmt.Sprintf("%s [bank %s]", e.Address, ref.Bank))
	} else {
		imgui.Text(e.Address)
	}
	imgui.PopStyleColor()

	imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
	imgui.Text(e.Operator)
	imgui.PopStyleColor()

	if e.Operand.String() != "" {
		imgui.SameLine()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
		imgui.Text(e.Operand.String())
		imgui.PopStyleColor()
	}

	// add HBLANK/VBLANK information
	if ref.IsHblank && ref.TV.VBlank {
		imguiSeparator()
		imguiLabel("[HBLANK/VBLANK]")
	} else if ref.IsHblank {
		imguiSeparator()
		imguiLabel("[HBLANK]")
	} else if ref.TV.VBlank {
		imguiSeparator()
		imguiLabel("[VBLANK]")
	}
}

func (win *winDbgScr) setCropping(set bool) {
	win.cropped = set
	win.createTextures = true
}

// resize() implements the textureRenderer interface.
func (win *winDbgScr) resize() {
	win.createTextures = true
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). must be inside
// screen critical section.
func (win *winDbgScr) render() {
	var pixels *image.RGBA
	var overlay *image.RGBA
	var phosphor *image.RGBA

	if win.cropped {
		if win.debugColors {
			pixels = win.scr.crit.cropElementPixels
		} else {
			pixels = win.scr.crit.cropPixels
		}
		overlay = win.scr.crit.cropOverlayPixels
		phosphor = win.scr.crit.cropPhosphor
	} else {
		if win.debugColors {
			pixels = win.scr.crit.elementPixels
		} else {
			pixels = win.scr.crit.pixels
		}
		overlay = win.scr.crit.overlayPixels
		phosphor = win.scr.crit.phosphor
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
			gl.Ptr(overlay.Pix))

		gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitDbgScr)
		gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(phosphor.Bounds().Size().X), int32(phosphor.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(phosphor.Pix))

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
			gl.Ptr(overlay.Pix))

		gl.ActiveTexture(gl.TEXTURE0 + phosphorTextureUnitDbgScr)
		gl.BindTexture(gl.TEXTURE_2D, win.phosphorTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(phosphor.Bounds().Size().X), int32(phosphor.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(phosphor.Pix))
	}

	// set screen image scaling (and image padding) based on the current window size
	win.setScaleAndPadding(win.contentDim)
}

// must be called from with a critical section.
func (win *winDbgScr) setScaleAndPadding(sz imgui.Vec2) {
	sz.Y -= win.toolbarHeight
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

// must be called from with a critical section.
func (win *winDbgScr) getScaledWidth(cropped bool) float32 {
	if cropped {
		return float32(win.scr.crit.cropPixels.Bounds().Size().X) * win.getScaling(true)
	}
	return float32(win.scr.crit.pixels.Bounds().Size().X) * win.getScaling(true)
}

// must be called from with a critical section.
func (win *winDbgScr) getScaledHeight(cropped bool) float32 {
	if cropped {
		return float32(win.scr.crit.cropPixels.Bounds().Size().Y) * win.getScaling(false)
	}
	return float32(win.scr.crit.pixels.Bounds().Size().Y) * win.getScaling(false)
}

func (win *winDbgScr) getScaling(horiz bool) float32 {
	if horiz {
		return pixelWidth * win.scr.aspectBias * win.scaling
	}
	return win.scaling
}
