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
	"github.com/inkyblackness/imgui-go/v4"
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

	// textures
	screenTexture   uint32
	elementsTexture uint32
	overlayTexture  uint32

	// the amount of alpha to apply to the overlay
	overlayAlpha float32

	// (re)create textures on next render()
	createTextures bool

	// how to present the screen in the window
	debugColors bool
	cropped     bool
	crt         bool

	// the tv screen has captured mouse input
	isCaptured bool

	// clocks and scanline equivalent position of the mouse
	mouseClock    int
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
	screenDim imgui.Vec2

	// the basic amount by which the image should be scaled. this value is
	// applie to the vertical axis directly. horizontal scaling is scaled by
	// pixelWidth and aspectBias also. use horizScaling() for that.
	scaling float32

	// the dimensions required for the combo widgets
	specComboDim imgui.Vec2
}

func newWinDbgScr(img *SdlImgui) (window, error) {
	win := &winDbgScr{
		img:          img,
		scr:          img.screen,
		scaling:      2.0,
		crt:          false,
		cropped:      true,
		overlayAlpha: 0.4,
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.screenTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.elementsTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.elementsTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.GenTextures(1, &win.overlayTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

// List of valid overlay reflection overlay types.
type overlay int

const (
	overlayNoOverlay overlay = iota
	overlayWSYNC
	overlayCollisions
	overlayHMOVE
	overlayRSYNC
	overlayCoprocessor
)

func (o overlay) String() string {
	switch o {
	case overlayWSYNC:
		return "WSYNC overlay"
	case overlayCollisions:
		return "Collisions overlay"
	case overlayHMOVE:
		return "HMOVE overlay"
	case overlayRSYNC:
		return "RSYNC overlay"
	case overlayCoprocessor:
		return "CoProcessor overlay"
	}
	return "No overlay"
}

func (win *winDbgScr) init() {
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
		w = win.scaledWidth(true)
		h = win.scaledHeight(true)
	} else {
		w = win.scaledWidth(false)
		h = win.scaledHeight(false)
	}

	// if isCaptured flag is set then change the title and border colors of the
	// TV Screen window.
	if win.isCaptured {
		imgui.PushStyleColor(imgui.StyleColorTitleBgActive, win.img.cols.CapturedScreenTitle)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.CapturedScreenBorder)
		defer imgui.PopStyleColorV(2)
	}

	imgui.SetNextWindowPosV(imgui.Vec2{8, 28}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{611, 470}, imgui.ConditionFirstUseEver)

	// we don't want to ever show scrollbars
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoScrollbar)

	// note size of remaining window and content area
	win.screenDim = imgui.ContentRegionAvail()
	win.screenDim.Y -= win.toolbarHeight

	// screen image, overlays, menus and tooltips
	imgui.BeginChildV("##image", imgui.Vec2{win.screenDim.X, win.screenDim.Y}, false, imgui.WindowFlagsNoScrollbar)

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// note the current cursor position. we'll use this to everything to the
	// corner of the screen.
	screenOrigin := imgui.CursorScreenPos()

	// push style info for screen and overlay ImageButton(). we're using
	// ImageButton because an Image will not capture mouse events and pass them
	// to the parent window. this means that a click-drag on the screen/overlay
	// will move the window, which we don't want.
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{0.0, 0.0})

	// screen texture
	imgui.SetCursorScreenPos(screenOrigin)
	imgui.ImageButton(imgui.TextureID(win.screenTexture), imgui.Vec2{w, h})

	// debug colors / element layer
	if win.debugColors {
		imgui.SetCursorScreenPos(screenOrigin)
		imgui.ImageButton(imgui.TextureID(win.elementsTexture), imgui.Vec2{w, h})
	}

	// overlay texture on top of screen texture
	imgui.SetCursorScreenPos(screenOrigin)
	imgui.ImageButton(imgui.TextureID(win.overlayTexture), imgui.Vec2{w, h})

	// pop style info for screen and overlay textures
	imgui.PopStyleVar()
	imgui.PopStyleColorV(3)

	// popup menu on right mouse button
	//
	// we only call OpenPopup() if it's not already open. also, care taken to
	// avoid menu opening when releasing a captured mouse.
	if !win.isCaptured && imgui.IsItemHovered() && imgui.IsMouseDown(1) {
		imgui.OpenPopup("breakMenu")
	}

	if imgui.BeginPopup("breakMenu") {
		imgui.Text("Break")
		imguiSeparator()
		if imgui.Selectable(fmt.Sprintf("Scanline=%d", win.mouseScanline)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d", win.mouseScanline))
		}
		if imgui.Selectable(fmt.Sprintf("Clock=%d", win.mouseClock)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK CL %d", win.mouseClock))
		}
		if imgui.Selectable(fmt.Sprintf("Scanline=%d & Clock=%d", win.mouseScanline, win.mouseClock)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d & CL %d", win.mouseScanline, win.mouseClock))
		}
		imgui.EndPopup()
	}

	// draw tool tip
	if imgui.IsWindowHovered() {
		win.drawReflectionTooltip(screenOrigin)
	}

	// accept mouse clicks if window is focused
	if imgui.IsWindowFocused() {
		// mouse click will cause the rewind goto coords to run only when the
		// emulation is paused
		if win.img.state == gui.StatePaused {
			if imgui.IsMouseReleased(0) {
				win.img.screen.gotoCoordsX = win.mouseClock
				win.img.screen.gotoCoordsY = win.img.wm.dbgScr.mouseScanline
				win.img.lz.Dbg.PushGotoCoords(win.img.lz.TV.Frame, win.mouseScanline, win.mouseClock-specification.ClksHBlank)
			}
		}
	}

	// end of screen image
	imgui.EndChild()

	// start of tool bar
	win.toolbarHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		win.drawCoordsLine()
		imgui.Spacing()

		// tv status line
		imgui.PushItemWidth(win.specComboDim.X)
		if imgui.BeginComboV("##spec", win.img.lz.TV.Spec.ID, imgui.ComboFlagsNoArrowButton) {
			for _, s := range specification.SpecList {
				if imgui.Selectable(s) {
					win.img.term.pushCommand(fmt.Sprintf("TV SPEC %s", s))
				}
			}
			imgui.EndCombo()
		}
		imgui.PopItemWidth()
		imgui.SameLineV(0, 15)

		// display toggles
		imgui.Checkbox("Debug Colours", &win.debugColors)
		imgui.SameLineV(0, 15)
		if imgui.Checkbox("Cropping", &win.cropped) {
			win.setCropping(win.cropped)
		}
		imgui.SameLineV(0, 15)
		imgui.Checkbox("CRT Effects", &win.crt)
		imgui.SameLine()

		imgui.SameLineV(0, 15)
		win.drawOverlayPopup()
	})

	imgui.End()
}

func (win *winDbgScr) drawOverlayPopup() {
	o := win.img.screen.crit.overlay.Load().(overlay)

	imgui.PushItemWidth(200)
	defer imgui.PopItemWidth()

	if imgui.Button(o.String()) {
		imgui.OpenPopup("overlaySelector")
	}

	if imgui.BeginPopup("overlaySelector") {
		if imgui.BeginTable("overlayTable", 2) {
			imgui.TableSetupColumnV("overlay", imgui.TableColumnFlagsWidthFixed, 200, 1)
			imgui.TableSetupColumnV("key", imgui.TableColumnFlagsWidthFixed, 200, 2)

			imgui.TableNextRow()

			// column 1
			imgui.TableNextColumn()
			v := int(o)

			// draw radio buttons and prepare for a test of whether any of them
			// have been pressed. note that the OR with ok happens after the
			// drawing of the button. if we order the condition the other way, the
			// test might short-circuit and the button will not be drawn until next
			// frame.
			ok := false
			ok = imgui.RadioButtonInt(overlayNoOverlay.String(), &v, int(overlayNoOverlay)) || ok
			ok = imgui.RadioButtonInt(overlayWSYNC.String(), &v, int(overlayWSYNC)) || ok
			ok = imgui.RadioButtonInt(overlayCollisions.String(), &v, int(overlayCollisions)) || ok
			ok = imgui.RadioButtonInt(overlayHMOVE.String(), &v, int(overlayHMOVE)) || ok
			ok = imgui.RadioButtonInt(overlayRSYNC.String(), &v, int(overlayRSYNC)) || ok

			// special handling for coprocessor layer
			if win.img.lz.CoProc.HasCoProcBus {
				ok = imgui.RadioButtonInt(win.img.lz.CoProc.ID, &v, int(overlayCoprocessor)) || ok
			} else if o == overlayCoprocessor {
				// there is no coprocessor so make sure the coprocessor option is
				// not selected - it won't be shown
				v = int(overlayNoOverlay)
			}

			// a radiobutton has been pressed
			if ok {
				// change overlay
				win.img.screen.crit.overlay.Store(overlay(v))

				// replot the overlay
				win.img.screen.replotOverlay()
			}

			// column 2
			imgui.TableNextColumn()
			imgui.Spacing()

			switch o {
			case overlayWSYNC:
				col := packedColSetAlpha(win.img.cols.reflectionColors[reflection.WSYNC], win.overlayAlpha)
				win.img.imguiColorLabel(col, "WSYNC active")
			case overlayCollisions:
				col := packedColSetAlpha(win.img.cols.reflectionColors[reflection.Collision], win.overlayAlpha)
				win.img.imguiColorLabel(col, "Collision")
				col = packedColSetAlpha(win.img.cols.reflectionColors[reflection.CXCLR], win.overlayAlpha)
				win.img.imguiColorLabel(col, "CXCLR")
			case overlayHMOVE:
				col := packedColSetAlpha(win.img.cols.reflectionColors[reflection.HMOVEdelay], win.overlayAlpha)
				win.img.imguiColorLabel(col, "Delay")
				col = packedColSetAlpha(win.img.cols.reflectionColors[reflection.HMOVEripple], win.overlayAlpha)
				win.img.imguiColorLabel(col, "Ripple")
				col = packedColSetAlpha(win.img.cols.reflectionColors[reflection.HMOVElatched], win.overlayAlpha)
				win.img.imguiColorLabel(col, "Latch")
			case overlayRSYNC:
				col := packedColSetAlpha(win.img.cols.reflectionColors[reflection.RSYNCalign], win.overlayAlpha)
				win.img.imguiColorLabel(col, "Align")
				col = packedColSetAlpha(win.img.cols.reflectionColors[reflection.RSYNCreset], win.overlayAlpha)
				win.img.imguiColorLabel(col, "Reset")
			case overlayCoprocessor:
				col := packedColSetAlpha(win.img.cols.reflectionColors[reflection.CoprocessorActive], win.overlayAlpha)
				win.img.imguiColorLabel(col, fmt.Sprintf("%s Active", win.img.lz.CoProc.ID))
			}

			imgui.EndTable()
		}

		// label for alpha slider
		var label string
		if win.overlayAlpha > 0.9 {
			label = "very opaque"
		} else if win.overlayAlpha > 0.6 {
			label = "opaque"
		} else if win.overlayAlpha > 0.3 {
			label = "faint"
		} else {
			label = "very faint"
		}

		// alpha slider
		imgui.Spacing()
		imguiLabel("Transparency")
		imgui.PushItemWidth(200.0)
		imgui.SliderFloatV("##alpha", &win.overlayAlpha, 0.1, 1.0, label, imgui.SliderFlagsNone)
		imgui.PopItemWidth()

		imgui.EndPopup()
	}
}

func (win *winDbgScr) drawCoordsLine() {
	imguiLabel("Frame:")
	imguiLabel(fmt.Sprintf("%-4d", win.img.lz.TV.Frame))
	imgui.SameLineV(0, 15)
	imguiLabel("Scanline:")
	if win.img.lz.TV.Scanline > 999 {
	} else {
		imguiLabel(fmt.Sprintf("%-3d", win.img.lz.TV.Scanline))
	}
	imgui.SameLineV(0, 15)
	imguiLabel("Clock:")
	imguiLabel(fmt.Sprintf("%-3d", win.img.lz.TV.Clock))

	// include tv signal information
	imgui.SameLineV(0, 20)
	imgui.Text(win.img.lz.TV.LastSignal.String())
}

// called from within a win.scr.crit.section Lock().
func (win *winDbgScr) drawReflectionTooltip(screenOrigin imgui.Vec2) {
	// get mouse position and transform
	mp := imgui.MousePos().Minus(screenOrigin)
	if win.cropped {
		sz := win.scr.crit.cropPixels.Bounds().Size()
		mp.X = mp.X / win.scaledWidth(true) * float32(sz.X)
		mp.Y = mp.Y / win.scaledHeight(true) * float32(sz.Y)
		mp.X += float32(specification.ClksHBlank)
		mp.Y += float32(win.scr.crit.topScanline)
	} else {
		sz := win.scr.crit.pixels.Bounds().Size()
		mp.X = mp.X / win.scaledWidth(false) * float32(sz.X)
		mp.Y = mp.Y / win.scaledHeight(false) * float32(sz.Y)
	}

	win.mouseClock = int(mp.X)
	win.mouseScanline = int(mp.Y)

	// get reflection information
	var ref reflection.VideoStep

	if win.mouseClock < len(win.scr.crit.reflection) && win.mouseScanline < len(win.scr.crit.reflection[win.mouseClock]) {
		ref = win.scr.crit.reflection[win.mouseClock][win.mouseScanline]
	}

	if win.isCaptured {
		return
	}

	// present tooltip showing pixel coords at a minimum
	imgui.BeginTooltip()
	defer imgui.EndTooltip()

	imgui.Text(fmt.Sprintf("Scanline: %d", win.mouseScanline))
	imgui.Text(fmt.Sprintf("Clock: %d", win.mouseClock-specification.ClksHBlank))

	switch win.scr.crit.overlay.Load().(overlay) {
	case overlayWSYNC:
		imguiSeparator()
		if ref.WSYNC {
			imgui.Text("6507 is not ready")
		} else {
			imgui.Text("6507 program is running")
		}
	case overlayCollisions:
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
	case overlayHMOVE:
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
	case overlayRSYNC:
		// no RSYNC specific hover information
	case overlayCoprocessor:
		imguiSeparator()
		if ref.CoprocessorActive {
			imgui.Text(fmt.Sprintf("%s is working", win.img.lz.CoProc.ID))
		} else {
			imgui.Text("6507 program is running")
		}
	case overlayNoOverlay:
		// no overlay

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
}

func (win *winDbgScr) setCropping(set bool) {
	win.cropped = set
	win.createTextures = true
}

// resize() implements the textureRenderer interface.
func (win *winDbgScr) resize() {
	win.createTextures = true

	// scaling is set every render() rather than on resize(), like it is done
	// in the playscr. this is because scaling needs to consider the size of
	// the dbgscr window which can change more often than resize() is called.
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). must be inside
// screen critical section.
func (win *winDbgScr) render() {
	var pixels *image.RGBA
	var elements *image.RGBA
	var overlay *image.RGBA

	if win.cropped {
		pixels = win.scr.crit.cropPixels
		elements = win.scr.crit.cropElementPixels
		overlay = win.scr.crit.cropOverlayPixels
	} else {
		pixels = win.scr.crit.pixels
		elements = win.scr.crit.elementPixels
		overlay = win.scr.crit.overlayPixels
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

		gl.BindTexture(gl.TEXTURE_2D, win.elementsTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(elements.Pix))

		gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0,
			gl.RGBA, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y), 0,
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(overlay.Pix))

		win.createTextures = false
	} else {
		gl.BindTexture(gl.TEXTURE_2D, win.screenTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(pixels.Pix))

		gl.BindTexture(gl.TEXTURE_2D, win.elementsTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(elements.Pix))

		gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0,
			0, 0, int32(pixels.Bounds().Size().X), int32(pixels.Bounds().Size().Y),
			gl.RGBA, gl.UNSIGNED_BYTE,
			gl.Ptr(overlay.Pix))
	}

	// set screen image scaling (and image padding) based on the current window
	// size. unlike the playscr we check and set scaling every render frame.
	win.setScaling()
}

// must be called from with a critical section.
func (win *winDbgScr) setScaling() {
	winAspectRatio := win.screenDim.X / win.screenDim.Y

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
		win.scaling = win.screenDim.Y / imageH
		win.imagePadding = imgui.Vec2{X: float32(int((win.screenDim.X - (imageW * win.scaling)) / 2))}
	} else {
		win.scaling = win.screenDim.X / imageW
		win.imagePadding = imgui.Vec2{Y: float32(int((win.screenDim.Y - (imageH * win.scaling)) / 2))}
	}
}

// must be called from with a critical section.
func (win *winDbgScr) scaledWidth(cropped bool) float32 {
	if cropped {
		return float32(win.scr.crit.cropPixels.Bounds().Size().X) * win.horizScaling()
	}
	return float32(win.scr.crit.pixels.Bounds().Size().X) * win.horizScaling()
}

// must be called from with a critical section.
func (win *winDbgScr) scaledHeight(cropped bool) float32 {
	if cropped {
		return float32(win.scr.crit.cropPixels.Bounds().Size().Y) * win.scaling
	}
	return float32(win.scr.crit.pixels.Bounds().Size().Y) * win.scaling
}

// for vertical scaling simply refer to the scaling field.
func (win *winDbgScr) horizScaling() float32 {
	return pixelWidth * win.scr.aspectBias * win.scaling
}
