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
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
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

	// scaled mouse coordinates. top-left corner is zero for uncropped screens.
	// cropped screens are adjusted as required
	//
	// use these values to index the reflection array, for example
	mouseX int
	mouseY int

	// mouse position adjusted so that clock and scanline represent the
	// underlying screen (taking cropped setting into account)
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

func (win *winDbgScr) init() {
	win.specComboDim = imguiGetFrameDim("", specification.SpecList...)
	win.overlayComboDim = imguiGetFrameDim("", reflection.OverlayLabels...)
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

const breakMenuPopupID = "dbgScreenBreakMenu"

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
	imgui.SetNextWindowSizeV(imgui.Vec2{637, 431}, imgui.ConditionFirstUseEver)

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

	// get mouse position if breakmenu is not open
	if !imgui.IsPopupOpen(breakMenuPopupID) {
		win.mousePos = imgui.MousePos().Minus(win.screenOrigin)

		// scaled mouse position coordinates
		win.mouseX = int(win.mousePos.X / win.xscaling)
		win.mouseY = int(win.mousePos.Y / win.yscaling)

		// corresponding clock and scanline values for scaled mouse coordinates
		win.mouseClock = win.mouseX
		win.mouseScanline = win.mouseY

		// adjust depending on whether screen is cropped (or in CRT Preview)
		if win.cropped || win.crtPreview {
			win.mouseX += specification.ClksHBlank
			win.mouseY += win.scr.crit.frameInfo.VisibleTop
			win.mouseScanline += win.scr.crit.frameInfo.VisibleTop
		} else {
			win.mouseClock -= specification.ClksHBlank
		}
	}

	// push style info for screen and overlay ImageButton(). we're using
	// ImageButton because an Image will not capture mouse events and pass them
	// to the parent window. this means that a click-drag on the screen/overlay
	// will move the window, which we don't want.
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{0.0, 0.0})

	imageHovered := false

	if win.crtPreview {
		imgui.ImageButton(imgui.TextureID(win.normalTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})
		imageHovered = imgui.IsItemHovered()
	} else {
		// choose which texture to use depending on whether elements is selected
		if win.elements {
			imgui.ImageButton(imgui.TextureID(win.elementsTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})
		} else {
			imgui.ImageButton(imgui.TextureID(win.normalTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})
		}
		imageHovered = imgui.IsItemHovered()

		// overlay texture on top of screen texture
		imgui.SetCursorScreenPos(win.screenOrigin)
		imgui.ImageButton(imgui.TextureID(win.overlayTexture), imgui.Vec2{win.scaledWidth, win.scaledHeight})

		// popup menu on right mouse button
		//
		// we only call OpenPopup() if it's not already open. also, care taken to
		// avoid menu opening when releasing a captured mouse.
		if !win.isCaptured && imgui.IsItemHovered() && imgui.IsMouseDown(1) {
			imgui.OpenPopup(breakMenuPopupID)
		}

		if imgui.BeginPopup(breakMenuPopupID) {
			imgui.Text("Break on TV Coords")
			imguiSeparator()
			if imgui.Selectable(fmt.Sprintf("Scanline %d", win.mouseScanline)) {
				win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d", win.mouseScanline))
			}
			if imgui.Selectable(fmt.Sprintf("Clock %d", win.mouseClock)) {
				win.img.term.pushCommand(fmt.Sprintf("BREAK CL %d", win.mouseClock))
			}
			if imgui.Selectable(fmt.Sprintf("Scanline %d & Clock %d", win.mouseScanline, win.mouseClock)) {
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
	if imgui.IsWindowFocused() && imageHovered {
		// mouse click will cause the rewind goto coords to run only when the
		// emulation is paused
		if win.img.emulation.State() == emulation.Paused {
			if imgui.IsMouseDown(0) {
				coords := coords.TelevisionCoords{
					Frame:    win.img.lz.TV.Coords.Frame,
					Scanline: win.mouseScanline,
					Clock:    win.mouseClock,
				}

				// if mouse is off the end of the screen then adjust the
				// scanline (we want to goto) to just before the end of the
				// screen (the actual end of the screen might be a half
				// scanline - this limiting effect is purely visual so accuracy
				// isn't paramount)
				if coords.Scanline >= win.img.screen.crit.frameInfo.TotalScanlines {
					coords.Scanline = win.img.screen.crit.frameInfo.TotalScanlines - 1
					if coords.Scanline < 0 {
						coords.Scanline = 0
					}
				}

				// match against the actual mouse scanline not the adjusted scanline
				if win.img.screen.gotoCoordsX != win.mouseClock || win.img.screen.gotoCoordsY != win.img.wm.dbgScr.mouseScanline {
					win.img.screen.gotoCoordsX = win.mouseClock
					win.img.screen.gotoCoordsY = win.img.wm.dbgScr.mouseScanline
					win.img.dbg.GotoCoords(coords)
				}
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
	if imgui.BeginComboV("##spec", win.img.lz.TV.FrameInfo.Spec.ID, imgui.ComboFlagsNone) {
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

	// TV coordinates
	imgui.Text(win.img.lz.TV.Coords.String())

	// tv signal information
	imgui.SameLineV(0, 20)
	imgui.Text(win.img.lz.TV.LastSignal.String())

	// unsynced indicator
	if !win.scr.crit.frameInfo.VSynced {
		imgui.SameLineV(0, 20)
		imgui.Text("UNSYNCED")
	}
}

func (win *winDbgScr) drawOverlayCombo() {
	imgui.PushItemWidth(win.overlayComboDim.X + imgui.FrameHeight())

	// change coprocessor text to CoProcID if a coprocessor is present
	v := win.img.screen.crit.overlay
	if v == reflection.OverlayLabels[reflection.OverlayCoproc] {
		if win.img.lz.Cart.HasCoProcBus {
			v = win.img.lz.Cart.CoProcID
		} else {
			// it's possible for the coprocessor overlay to be selected and
			// then a different ROM loaded that has no coprocessor. in this
			// case change the overlay to none.
			win.img.screen.crit.overlay = reflection.OverlayLabels[reflection.OverlayNone]
		}
	}

	if imgui.BeginComboV("##overlay", v, imgui.ComboFlagsNone) {
		for i, s := range reflection.OverlayLabels {
			// special handling for coprocesor bus - only show it if a
			// coprocessor is present
			if i != int(reflection.OverlayCoproc) {
				if imgui.Selectable(s) {
					win.img.screen.crit.overlay = s
					win.img.screen.plotOverlay()
				}
			} else if win.img.lz.Cart.HasCoProcBus {
				// if ROM has a coprocessor change the option label to the
				// appropriate coprocessor ID
				if imgui.Selectable(win.img.lz.Cart.CoProcID) {
					// we still store the "Coprocessor" string and not the ID
					// string. this way we don't need any fancy conditions
					// elsewhere
					win.img.screen.crit.overlay = s
					win.img.screen.plotOverlay()
				}
			}
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
}

func (win *winDbgScr) drawOverlayColorKey() {
	switch win.img.screen.crit.overlay {
	case reflection.OverlayLabels[reflection.OverlayWSYNC]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("WSYNC", win.img.cols.reflectionColors[reflection.WSYNC])
	case reflection.OverlayLabels[reflection.OverlayCollision]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("Collision", win.img.cols.reflectionColors[reflection.Collision])
		imgui.SameLineV(0, 15)
		imguiColorLabel("CXCLR", win.img.cols.reflectionColors[reflection.CXCLR])
	case reflection.OverlayLabels[reflection.OverlayHMOVE]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("Delay", win.img.cols.reflectionColors[reflection.HMOVEdelay])
		imgui.SameLineV(0, 15)
		imguiColorLabel("Ripple", win.img.cols.reflectionColors[reflection.HMOVEripple])
		imgui.SameLineV(0, 15)
		imguiColorLabel("Latch", win.img.cols.reflectionColors[reflection.HMOVElatched])
	case reflection.OverlayLabels[reflection.OverlayRSYNC]:
		imgui.SameLineV(0, 20)
		imguiColorLabel("Align", win.img.cols.reflectionColors[reflection.RSYNCalign])
		imgui.SameLineV(0, 15)
		imguiColorLabel("Reset", win.img.cols.reflectionColors[reflection.RSYNCreset])
	case reflection.OverlayLabels[reflection.OverlayCoproc]:
		imgui.SameLineV(0, 20)

		// display text includes coprocessor ID
		key := fmt.Sprintf("%s Active", win.img.lz.Cart.CoProcID)
		imguiColorLabel(key, win.img.cols.reflectionColors[reflection.CoprocessorActive])
	}
}

// called from within a win.scr.crit.section Lock().
func (win *winDbgScr) drawReflectionTooltip() {
	if win.isCaptured {
		return
	}

	// outside bounds of window
	if win.mousePos.X < 0.0 || win.mousePos.Y < 0.0 {
		return
	}

	mouseOffset := win.mouseX + win.mouseY*specification.ClksScanline

	if mouseOffset < 0 || mouseOffset > len(win.scr.crit.reflection) {
		return
	}

	// get reflection information
	ref := win.scr.crit.reflection[mouseOffset]

	// draw tooltip
	imguiTooltip(func() {
		imgui.Text(fmt.Sprintf("Scanline: %d", win.mouseScanline))
		imgui.Text(fmt.Sprintf("Clock: %d", win.mouseClock))

		e := win.img.dbg.Disasm.FormatResult(ref.Bank, ref.CPU, disassembly.EntryLevelBlessed)
		if e.Address == "" {
			return
		}

		imguiSeparator()

		// if mouse is over a pixel from the previous frame then show nothing except a note
		if win.img.emulation.State() == emulation.Paused {
			if win.mouseScanline > win.img.screen.crit.lastScanline ||
				(win.mouseScanline == win.img.screen.crit.lastScanline && win.mouseClock > win.img.screen.crit.lastClock) {
				imgui.Text("From previous frame")
				imguiSeparator()
			}
		}

		// pixel swatch. using black swatch if pixel is HBLANKed or VBLANKed
		_, _, pal := win.img.imguiTVPalette()
		px := signal.ColorSignal((ref.Signal & signal.Color) >> signal.ColorShift)
		if ref.IsHblank || ref.Signal&signal.VBlank == signal.VBlank || px == signal.VideoBlack {
			imguiColorLabel("No color signal", pal[0])
		} else {
			// not using GetColor() function. arguably we should but we've
			// protected the array access with the VideoBlack test above.
			imguiColorLabel(ref.VideoElement.String(), pal[px])
		}

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

		switch win.scr.crit.overlay {
		case reflection.OverlayLabels[reflection.OverlayNone]:
			s := ref.Signal.String()
			if ref.IsHblank && len(s) > 0 {
				imguiSeparator()
				imgui.Text("HBLANK")
				imgui.SameLine()
				imgui.Text(s)
			} else if ref.IsHblank {
				imguiSeparator()
				imgui.Text("HBLANK")
			} else if len(s) > 0 {
				imguiSeparator()
				imgui.Text(s)
			}
		case reflection.OverlayLabels[reflection.OverlayWSYNC]:
			imguiSeparator()
			if ref.WSYNC {
				imgui.Text("6507 is not ready")
			} else {
				imgui.Text("6507 program is running")
			}
		case reflection.OverlayLabels[reflection.OverlayCollision]:
			imguiSeparator()

			imguiLabel("CXM0P ")
			drawRegister("##CXM0P", win.img.lz.Collisions.CXM0P, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXM1P ")
			drawRegister("##CXM1P", win.img.lz.Collisions.CXM1P, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXP0FB")
			drawRegister("##CXP0FB", win.img.lz.Collisions.CXP0FB, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXP1FB")
			drawRegister("##CXP1FB", win.img.lz.Collisions.CXP1FB, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXM0FB")
			drawRegister("##CXM0FB", win.img.lz.Collisions.CXM0FB, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXM1FB")
			drawRegister("##CXM1FB", win.img.lz.Collisions.CXM1FB, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXBLPF")
			drawRegister("##CXBLPF", win.img.lz.Collisions.CXBLPF, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXPPMM")
			drawRegister("##CXPPMM", win.img.lz.Collisions.CXPPMM, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)

			imguiSeparator()

			s := ref.Collision.LastVideoCycle.String()
			if s != "" {
				imgui.Text(s)
			} else {
				imgui.Text("no new collision")
			}
		case reflection.OverlayLabels[reflection.OverlayHMOVE]:
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
		case reflection.OverlayLabels[reflection.OverlayRSYNC]:
			// no RSYNC specific hover information
		case reflection.OverlayLabels[reflection.OverlayCoproc]:
			imguiSeparator()
			if ref.CoprocessorActive {
				imgui.Text(fmt.Sprintf("%s is working", win.img.lz.Cart.CoProcID))
			} else {
				imgui.Text("6507 program is running")
			}
		}
	}, false)
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
	adjW := w * pixelWidth * win.scr.crit.frameInfo.Spec.AspectBias

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
	win.xscaling = scaling * pixelWidth * win.scr.crit.frameInfo.Spec.AspectBias
	win.scaledWidth = w * win.xscaling
	win.scaledHeight = h * win.yscaling

	// get numscanlines while we're in critical section
	win.numScanlines = win.scr.crit.frameInfo.VisibleBottom - win.scr.crit.frameInfo.VisibleTop
}
