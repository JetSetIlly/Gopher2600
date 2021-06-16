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

	// (re)create textures on next render()
	createTextures bool

	// textures
	normalTexture   uint32
	elementsTexture uint32
	overlayTexture  uint32

	// how to present the screen in the window
	elements bool
	cropped  bool

	// the tv screen has captured mouse input
	isCaptured bool

	// imgui coords of mouse
	mousePos imgui.Vec2

	// clocks and scanline equivalent position of the mouse
	mouseClock    int
	mouseScanline int

	// height of tool bar at bottom of window. valid after first frame.
	toolbarHeight float32

	// additional padding for the image so that it is centred in its content space
	imagePadding imgui.Vec2

	// size of area available to the screen image and origin (position) of
	// image on the screen
	screenRegion imgui.Vec2
	screenOrigin imgui.Vec2

	// scaling of texture and calculated dimensions
	xscaling     float32
	yscaling     float32
	scaledWidth  float32
	scaledHeight float32

	// the dimensions required for the combo widgets
	specComboDim    imgui.Vec2
	overlayComboDim imgui.Vec2

	// number of scanlines in current image. taken from screen but is crit section safe
	numScanlines int

	// crtPreview option is special. it overrides the other options in the dbgScr to
	// show an uncropped CRT preview in the dbgscr window.
	crtPreview bool
}

func newWinDbgScr(img *SdlImgui) (window, error) {
	win := &winDbgScr{
		img:        img,
		scr:        img.screen,
		crtPreview: false,
		cropped:    true,
	}

	// set texture, creation of textures will be done after every call to resize()
	gl.GenTextures(1, &win.normalTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.normalTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.GenTextures(1, &win.elementsTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.elementsTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	gl.GenTextures(1, &win.overlayTexture)
	gl.BindTexture(gl.TEXTURE_2D, win.overlayTexture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

	return win, nil
}

// list of overlay labels.
var overlayLabels = []string{"No overlay", "WSYNC", "Collisions", "HMOVE", "RSYNC", "Coprocessor"}

// named indexes for overlay labels list.
const (
	overlayNone = iota
	overlayWSYNC
	overlayCollisions
	overlayHMOVE
	overlayRSYNC
	overlayCoprocessor
)

func (win *winDbgScr) init() {
	win.specComboDim = imguiGetFrameDim("", specification.SpecList...)
	win.overlayComboDim = imguiGetFrameDim("", overlayLabels...)
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

	// set screen image scaling (and image padding) based on the current window
	// size. unlike the playscr we check and set scaling every frame. we also
	// do this at draw() time rather than render() time, otherwise the sizing
	// would be a frame behind.
	win.setScaling()

	// if isCaptured flag is set then change the title and border colors of the
	// TV Screen window.
	if win.isCaptured {
		imgui.PushStyleColor(imgui.StyleColorTitleBgActive, win.img.cols.CapturedScreenTitle)
		imgui.PushStyleColor(imgui.StyleColorBorder, win.img.cols.CapturedScreenBorder)
		defer imgui.PopStyleColorV(2)
	}

	imgui.SetNextWindowPosV(imgui.Vec2{8, 28}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{632, 450}, imgui.ConditionFirstUseEver)

	// we don't want to ever show scrollbars
	imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoScrollbar)

	// note size of remaining window and content area
	win.screenRegion = imgui.ContentRegionAvail()
	win.screenRegion.Y -= win.toolbarHeight

	// screen image, overlays, menus and tooltips
	imgui.BeginChildV("##image", imgui.Vec2{win.screenRegion.X, win.screenRegion.Y}, false, imgui.WindowFlagsNoScrollbar)

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// note the current cursor position. we'll use this to everything to the
	// corner of the screen.
	win.screenOrigin = imgui.CursorScreenPos()

	// get mouse position and transform
	win.mousePos = imgui.MousePos().Minus(win.screenOrigin)

	// scale mouse position
	win.mouseClock = int(win.mousePos.X / win.xscaling)
	win.mouseScanline = int(win.mousePos.Y / win.yscaling)

	// adjust if image cropped or crt preview is active
	if win.cropped || win.crtPreview {
		win.mouseClock += specification.ClksHBlank
		win.mouseScanline += win.scr.crit.topScanline
	}

	// push style info for screen and overlay ImageButton(). we're using
	// ImageButton because an Image will not capture mouse events and pass them
	// to the parent window. this means that a click-drag on the screen/overlay
	// will move the window, which we don't want.
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{0.0, 0.0})

	if win.crtPreview {
		imgui.ImageButton(imgui.TextureID(win.normalTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})
	} else {
		// choose which texture to use depending on whether elements is selected
		if win.elements {
			imgui.ImageButton(imgui.TextureID(win.elementsTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})
		} else {
			imgui.ImageButton(imgui.TextureID(win.normalTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})
		}

		// overlay texture on top of screen texture
		imgui.SetCursorScreenPos(win.screenOrigin)
		imgui.ImageButton(imgui.TextureID(win.overlayTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})

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
		if imgui.IsItemHovered() {
			win.drawReflectionTooltip()
		}
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

	// pop style info for screen and overlay textures
	imgui.PopStyleVar()
	imgui.PopStyleColorV(3)

	// end of screen image
	imgui.EndChild()

	// start of tool bar
	win.toolbarHeight = imguiMeasureHeight(func() {
		// status line
		imgui.Spacing()
		win.drawCoordsLine()

		// options line
		imgui.Spacing()
		imgui.Spacing()

		// tv spec
		win.drawSpecCombo()

		// scaling indicator
		imgui.SameLineV(0, 15)
		imgui.AlignTextToFramePadding()
		imgui.Text(fmt.Sprintf("%.1fx", win.yscaling))

		// crt preview affects which debugging toggles are visible
		imgui.SameLineV(0, 15)
		if imgui.Checkbox("CRT Preview", &win.crtPreview) {
			win.createTextures = true
		}

		// debugging toggles
		if win.crtPreview {
			imgui.SameLineV(0, 15)
			imgui.AlignTextToFramePadding()
			imgui.Text("(using current CRT preferences)")
		} else {
			imgui.SameLineV(0, 15)
			imgui.Checkbox("Debug Colours", &win.elements)

			imgui.SameLineV(0, 15)
			if imgui.Checkbox("Cropping", &win.cropped) {
				win.createTextures = true
			}

			imgui.SameLineV(0, 15)
			win.drawOverlayCombo()
			win.drawOverlayColorKey()
		}
	})

	imgui.End()
}

func (win *winDbgScr) drawSpecCombo() {
	imgui.PushItemWidth(win.specComboDim.X + imgui.FrameHeight())
	if imgui.BeginComboV("##spec", win.img.lz.TV.Spec.ID, imgui.ComboFlagsNone) {
		for _, s := range specification.SpecList {
			if imgui.Selectable(s) {
				win.img.term.pushCommand(fmt.Sprintf("TV SPEC %s", s))
			}
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
	imgui.SameLineV(0, 15)
}

func (win *winDbgScr) drawCoordsLine() {
	imgui.Text("")
	imgui.SameLineV(0, 15)

	imgui.Text("Frame:")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%-4d", win.img.lz.TV.Frame))

	imgui.SameLineV(0, 15)
	imgui.Text("Scanline:")
	imgui.SameLine()
	if win.img.lz.TV.Scanline > 999 {
	} else {
		imgui.Text(fmt.Sprintf("%-3d", win.img.lz.TV.Scanline))
	}
	imgui.SameLineV(0, 15)
	imgui.Text("Clock:")
	imgui.SameLine()
	imgui.Text(fmt.Sprintf("%-3d", win.img.lz.TV.Clock))

	// include tv signal information
	imgui.SameLineV(0, 20)
	imgui.Text(win.img.lz.TV.LastSignal.String())

	// unsynced
	if !win.scr.crit.synced {
		imgui.SameLineV(0, 20)
		imgui.Text("UNSYNCED")
	}
}

func (win *winDbgScr) drawOverlayCombo() {
	imgui.PushItemWidth(win.overlayComboDim.X + imgui.FrameHeight())

	// change coprocessor text to CoProc.ID if a coprocessor is present
	v := win.img.screen.crit.overlay
	if v == overlayLabels[overlayCoprocessor] {
		if win.img.lz.CoProc.HasCoProcBus {
			v = win.img.lz.CoProc.ID
		} else {
			// it's possible for the coprocessor overlay to be selected and
			// then a different ROM loaded that has no coprocessor. in this
			// case change the overlay to none.
			win.img.screen.crit.overlay = overlayLabels[overlayNone]
		}
	}

	if imgui.BeginComboV("##overlay", v, imgui.ComboFlagsNone) {
		for i, s := range overlayLabels {
			if i != overlayCoprocessor {
				if imgui.Selectable(s) {
					win.img.screen.crit.overlay = s
					win.img.screen.replotOverlay()
				}
			} else if win.img.lz.CoProc.HasCoProcBus {
				// if ROM has a coprocessor change the optino label to the
				// appropriate ID
				if imgui.Selectable(win.img.lz.CoProc.ID) {
					// we still store the "Coprocessor" string and not the ID
					// string. this way we don't need any fancy conditions
					// elsewhere
					win.img.screen.crit.overlay = s
					win.img.screen.replotOverlay()
				}
			}
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
}

func (win *winDbgScr) drawOverlayColorKey() {
	switch win.img.screen.crit.overlay {
	case overlayLabels[overlayWSYNC]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("WSYNC", win.img.cols.reflectionColors[reflection.WSYNC])
	case overlayLabels[overlayCollisions]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("Collision", win.img.cols.reflectionColors[reflection.Collision])
		imgui.SameLineV(0, 15)
		imguiColorLabel("CXCLR", win.img.cols.reflectionColors[reflection.CXCLR])
	case overlayLabels[overlayHMOVE]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("Delay", win.img.cols.reflectionColors[reflection.HMOVEdelay])
		imgui.SameLineV(0, 15)
		imguiColorLabel("Ripple", win.img.cols.reflectionColors[reflection.HMOVEripple])
		imgui.SameLineV(0, 15)
		imguiColorLabel("Latch", win.img.cols.reflectionColors[reflection.HMOVElatched])
	case overlayLabels[overlayRSYNC]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("Align", win.img.cols.reflectionColors[reflection.RSYNCalign])
		imgui.SameLineV(0, 15)
		imguiColorLabel("Reset", win.img.cols.reflectionColors[reflection.RSYNCreset])
	case overlayLabels[overlayCoprocessor]:
		imgui.SameLineV(0, 20)

		// display text includes coprocessor ID
		key := fmt.Sprintf("%s Active", win.img.lz.CoProc.ID)
		imguiColorLabel(key, win.img.cols.reflectionColors[reflection.CoprocessorActive])
	}
}

// called from within a win.scr.crit.section Lock().
func (win *winDbgScr) drawReflectionTooltip() {
	if win.isCaptured {
		return
	}

	// lower boundary check
	if win.mousePos.X < 0.0 || win.mousePos.Y < 0.0 {
		return
	}

	// upper boundary check
	if win.mouseClock >= len(win.scr.crit.reflection) || win.mouseScanline >= len(win.scr.crit.reflection[win.mouseClock]) {
		return
	}

	// get reflection information
	ref := win.scr.crit.reflection[win.mouseClock][win.mouseScanline]

	// present tooltip showing pixel coords at a minimum
	imgui.BeginTooltip()
	defer imgui.EndTooltip()

	imgui.Text(fmt.Sprintf("Scanline: %d", win.mouseScanline))
	imgui.Text(fmt.Sprintf("Clock: %d", win.mouseClock-specification.ClksHBlank))

	switch win.scr.crit.overlay {
	case overlayLabels[overlayWSYNC]:
		imguiSeparator()
		if ref.WSYNC {
			imgui.Text("6507 is not ready")
		} else {
			imgui.Text("6507 program is running")
		}
	case overlayLabels[overlayCollisions]:
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
	case overlayLabels[overlayHMOVE]:
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
	case overlayLabels[overlayRSYNC]:
		// no RSYNC specific hover information
	case overlayLabels[overlayCoprocessor]:
		imguiSeparator()
		if ref.CoprocessorActive {
			imgui.Text(fmt.Sprintf("%s is working", win.img.lz.CoProc.ID))
		} else {
			imgui.Text("6507 program is running")
		}
	case overlayLabels[overlayNone]:
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

	if win.cropped || win.crtPreview {
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

	if win.createTextures {
		gl.BindTexture(gl.TEXTURE_2D, win.normalTexture)
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
		gl.BindTexture(gl.TEXTURE_2D, win.normalTexture)
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
}

// must be called from with a critical section.
func (win *winDbgScr) setScaling() {
	var w float32
	var h float32

	if win.cropped || win.crtPreview {
		w = float32(win.scr.crit.cropPixels.Bounds().Size().X)
		h = float32(win.scr.crit.cropPixels.Bounds().Size().Y)
	} else {
		w = float32(win.scr.crit.pixels.Bounds().Size().X)
		h = float32(win.scr.crit.pixels.Bounds().Size().Y)
	}
	adjW := w * pixelWidth * win.scr.crit.spec.AspectBias

	var scaling float32

	winRatio := win.screenRegion.X / win.screenRegion.Y
	aspectRatio := adjW / h

	if aspectRatio < winRatio {
		// window wider than TV screen
		scaling = win.screenRegion.Y / h
	} else {
		// TV screen wider than window
		scaling = win.screenRegion.X / adjW
	}

	// limit scaling to 1x
	if scaling < 1 {
		scaling = 1
	}

	win.imagePadding = imgui.Vec2{
		X: float32(int((win.screenRegion.X - (adjW * scaling)) / 2)),
		Y: float32(int((win.screenRegion.Y - (h * scaling)) / 2)),
	}

	win.yscaling = scaling
	win.xscaling = scaling * pixelWidth * win.scr.crit.spec.AspectBias
	win.scaledWidth = w * win.xscaling
	win.scaledHeight = h * win.yscaling

	// get numscanlines while we're in critical section
	win.numScanlines = win.scr.crit.bottomScanline - win.scr.crit.topScanline
}
