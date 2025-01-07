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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/reflection"
)

const winDbgScrID = "TV Screen"

type winDbgScr struct {
	debuggerWin

	img *SdlImgui

	// reference to screen data
	scr *screen

	// textures
	displayTexture  texture
	elementsTexture texture
	overlayTexture  texture

	// how to present the screen in the window
	elements bool
	cropped  bool

	// the tv screen has captured mouse input
	isCaptured bool

	// the current position of the actual mouse (not reliable after the frame it
	// was captured on)
	mouse dbgScrMouse

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

	// magnification fields
	magnifyTooltip dbgScrMagnifyTooltip
	magnifyWindow  dbgScrMagnifyWindow

	// whether mouse is hovering over screen image
	mouseHover bool
}

func newWinDbgScr(img *SdlImgui) (window, error) {
	win := &winDbgScr{
		img:     img,
		scr:     img.screen,
		cropped: true,
		magnifyTooltip: dbgScrMagnifyTooltip{
			zoom: magnifyDef,
		},
		magnifyWindow: dbgScrMagnifyWindow{
			zoom: magnifyDef,
		},
	}
	win.debuggerGeom.noFocusTracking = true

	// set texture, creation of textures will be done after every call to resize()
	win.displayTexture = img.rnd.addTexture(shaderDbgScr, true, true)
	win.overlayTexture = img.rnd.addTexture(shaderDbgScrOverlay, false, false)
	win.elementsTexture = img.rnd.addTexture(shaderDbgScr, true, true)
	win.magnifyTooltip.texture = img.rnd.addTexture(shaderColor, false, false)
	win.magnifyWindow.texture = img.rnd.addTexture(shaderColor, false, false)

	// call setScaling() now so that render() has something to work with - even
	// though setScaling() is called every draw if the window is open it will
	// leave render() nothing to work with if it isn't open on startup
	win.setScaling()

	return win, nil
}

func (win *winDbgScr) init() {
	win.specComboDim = imguiGetFrameDim("", specification.SpecList...)
	win.overlayComboDim = imguiGetFrameDim("", reflection.OverlayLabels...)
}

func (win *winDbgScr) id() string {
	return winDbgScrID
}

const breakMenuPopupID = "dbgScreenBreakMenu"

func (win *winDbgScr) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
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

	imgui.SetNextWindowPosV(imgui.Vec2{X: 8, Y: 28}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 637, Y: 431}, imgui.ConditionFirstUseEver)

	// we don't want to ever show scrollbars
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNoScrollbar) {
		win.draw()
	}
	win.debuggerGeom.update()
	imgui.End()

	// draw magnify window
	win.magnifyWindow.draw(win.img.cols)

	return true
}

func (win *winDbgScr) draw() {
	// note size of remaining window and content area
	win.screenRegion = imgui.ContentRegionAvail()
	win.screenRegion.Y -= win.toolbarHeight

	// screen image, overlays, menus and tooltips
	imgui.BeginChildV("##image", imgui.Vec2{X: win.screenRegion.X, Y: win.screenRegion.Y}, false, imgui.WindowFlagsNoScrollbar)

	// add horiz/vert padding around screen image
	imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

	// note the current cursor position. we'll use this to everything to the
	// corner of the screen.
	win.screenOrigin = imgui.CursorScreenPos()

	// get mouse position if breakmenu is not open
	if !imgui.IsPopupOpen(breakMenuPopupID) {
		win.mouse = win.currentMouse()
	}

	// push style info for screen and overlay ImageButton(). we're using
	// ImageButton because an Image will not capture mouse events and pass them
	// to the parent window. this means that a click-drag on the screen/overlay
	// will move the window, which we don't want.
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{X: 0.0, Y: 0.0})

	imgui.PushStyleColor(imgui.StyleColorDragDropTarget, win.img.cols.Transparent)

	if win.elements {
		imgui.ImageButton(imgui.TextureID(win.elementsTexture.getID()), imgui.Vec2{X: win.scaledWidth, Y: win.scaledHeight})
	} else {
		imgui.ImageButton(imgui.TextureID(win.displayTexture.getID()), imgui.Vec2{X: win.scaledWidth, Y: win.scaledHeight})
	}

	win.mouseHover = imgui.IsItemHovered()

	win.paintDragAndDrop()
	imgui.PopStyleColor()

	imageHovered := imgui.IsItemHovered()

	// overlay texture on top of screen texture
	imgui.SetCursorScreenPos(win.screenOrigin)
	imgui.ImageButton(imgui.TextureID(win.overlayTexture.getID()), imgui.Vec2{X: win.scaledWidth, Y: win.scaledHeight})

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
		if imgui.Selectable(fmt.Sprintf("Scanline %d", win.mouse.tv.Scanline)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d", win.mouse.tv.Scanline))
		}
		if imgui.Selectable(fmt.Sprintf("Clock %d", win.mouse.tv.Clock)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK CL %d", win.mouse.tv.Clock))
		}
		if imgui.Selectable(fmt.Sprintf("Scanline %d & Clock %d", win.mouse.tv.Scanline, win.mouse.tv.Clock)) {
			win.img.term.pushCommand(fmt.Sprintf("BREAK SL %d & CL %d", win.mouse.tv.Scanline, win.mouse.tv.Clock))
		}
		imguiSeparator()
		if imgui.Selectable(fmt.Sprintf("%c Magnify in Window", fonts.MagnifyingGlass)) {
			win.magnifyWindow.open = true
			win.magnifyWindow.setClipCenter(win.mouse)
		}
		imgui.EndPopup()
	}

	// draw tool tip
	if imgui.IsItemHovered() {
		win.drawReflectionTooltip()
	}

	// if mouse is over tv image then accept mouse clicks
	// . middle mouse button will control zoom window
	// . left button button will control rewinding of frame when emulation is paused
	if imageHovered {
		if imgui.IsMouseDown(2) {
			if win.magnifyWindow.open {
				win.magnifyWindow.setClipCenter(win.mouse)
			} else if imgui.IsMouseDoubleClicked(2) {
				win.magnifyWindow.open = true
				win.magnifyWindow.setClipCenter(win.mouse)
			}
		} else {
			if imgui.IsWindowFocused() {
				// mouse click will cause the rewind goto coords to run only when the
				// emulation is paused
				if win.img.dbg.State() == govern.Paused {
					if imgui.IsMouseDown(0) {
						coords := coords.TelevisionCoords{
							Frame:    win.img.cache.TV.GetCoords().Frame,
							Scanline: win.mouse.tv.Scanline,
							Clock:    win.mouse.tv.Clock,
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

						// match against the actual mouse.tv.Scanline not the adjusted scanline
						if win.img.screen.gotoCoordsX != win.mouse.tv.Clock || win.img.screen.gotoCoordsY != win.img.wm.dbgScr.mouse.tv.Scanline {
							win.img.screen.gotoCoordsX = win.mouse.tv.Clock
							win.img.screen.gotoCoordsY = win.img.wm.dbgScr.mouse.tv.Scanline
							win.img.dbg.GotoCoords(coords)
						}
					}
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

		// imgui.SetCursorPos(imgui.CursorPos().Plus(imgui.Vec2{X: win.imagePadding.X, Y: 0.0}))
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

		// debugging toggles
		imgui.SameLineV(0, 15)
		imgui.Checkbox("Debug Colours", &win.elements)

		imgui.SameLineV(0, 15)
		if imgui.Checkbox("Cropping", &win.cropped) {
			win.resize()
		}

		imgui.SameLineV(0, 15)
		win.drawOverlayCombo()
		win.drawOverlayComboTooltip()

		if win.img.screen.crit.overlay == reflection.OverlayLabels[reflection.OverlayNone] {
			imgui.SameLineV(0, 15)
			imgui.Checkbox("Magnify on hover", &win.magnifyTooltip.showInTooltip)
			win.img.imguiTooltipSimple(fmt.Sprintf("Show magnification in tooltip\n%c Mouse wheel to adjust zoom", fonts.Mouse))
		}
	})
}

func (win *winDbgScr) drawSpecCombo() {
	spec := win.img.cache.TV.GetFrameInfo().Spec.ID

	// special handling for PAL60 selection. PAL60 isn't a real TV spec and will
	// be treated as PAL by the television. however it is an option that can be
	// selected and it would seem odd if the selection was reflected by the user
	// interface
	if spec == "PAL" && win.img.cache.TV.GetReqSpecID() == "PAL60" {
		spec = "PAL60"
	}

	imgui.PushItemWidth(win.specComboDim.X + imgui.FrameHeight())
	if imgui.BeginComboV("##spec", spec, imgui.ComboFlagsNone) {
		for _, s := range specification.ReqSpecList {
			if s != "AUTO" {
				if imgui.Selectable(s) {
					win.img.term.pushCommand(fmt.Sprintf("TV SPEC %s", s))
				}
			}
		}
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		auto := win.img.cache.TV.GetReqSpecID() == "AUTO"
		if imgui.Checkbox("Auto", &auto) {
			if auto {
				win.img.term.pushCommand("TV SPEC AUTO")
			} else {
				s := win.img.cache.TV.GetFrameInfo().Spec.ID
				win.img.term.pushCommand(fmt.Sprintf("TV SPEC %s", s))
			}
			imgui.CloseCurrentPopup()
		}
		imgui.EndCombo()
	}
	imgui.PopItemWidth()
	imgui.SameLineV(0, 15)
}

func (win *winDbgScr) drawCoordsLine() {
	flgs := imgui.TableFlagsSizingFixedFit
	flgs |= imgui.TableFlagsBordersInnerV
	if imgui.BeginTableV("tvcoords", 5, imgui.TableFlagsSizingFixedFit, imgui.Vec2{X: 0.0, Y: 0.0}, 0.0) {
		imgui.TableSetupColumnV("##tvcoords_icon", imgui.TableColumnFlagsNone, imguiTextWidth(2), 0)
		imgui.TableSetupColumnV("##tvcoords_frame", imgui.TableColumnFlagsNone, imguiTextWidth(10), 1)
		imgui.TableSetupColumnV("##tvcoords_scanline", imgui.TableColumnFlagsNone, imguiTextWidth(13), 2)
		imgui.TableSetupColumnV("##tvcoords_clock", imgui.TableColumnFlagsNone, imguiTextWidth(10), 3)

		imgui.TableNextRow()

		imgui.TableNextColumn()
		imgui.Text(string(fonts.TV))

		// show geometry tooltip if this isn't frame zero
		frameInfo := win.img.screen.crit.frameInfo
		if frameInfo.FrameNum != 0 || win.img.cache.TV.GetCoords().Frame != 0 {
			win.img.imguiTooltip(func() {
				frameInfo := win.img.screen.crit.frameInfo
				flgs := imgui.TableFlagsSizingFixedFit

				imgui.Text("TV Screen Geometry")
				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()

				if imgui.BeginTableV("geometry_tooltip", 2, flgs, imgui.Vec2{X: 0.0, Y: 0.0}, 0.0) {
					imgui.TableSetupColumnV("##geometry_tooltip_desc", imgui.TableColumnFlagsNone, imguiTextWidth(9), 0)
					imgui.TableSetupColumnV("##geometry_tooltip_val", imgui.TableColumnFlagsNone, imguiTextWidth(3), 1)

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.Text("Scanlines")
					imgui.TableNextColumn()
					imgui.Text(fmt.Sprintf("%d", frameInfo.TotalScanlines))

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.Text("Top")
					imgui.TableNextColumn()
					imgui.Text(fmt.Sprintf("%d", frameInfo.VisibleTop))

					imgui.TableNextRow()
					imgui.TableNextColumn()
					imgui.Text("Bottom")
					imgui.TableNextColumn()
					imgui.Text(fmt.Sprintf("%d", frameInfo.VisibleBottom))

					imgui.EndTable()
				}

				imgui.Spacing()
				imgui.Separator()
				imgui.Spacing()

				imgui.Text(fmt.Sprintf("for Frame %d", frameInfo.FrameNum))
			}, true)
		}

		coords := win.img.cache.TV.GetCoords()

		imgui.TableNextColumn()
		imgui.Text(fmt.Sprintf("Frame: %d", coords.Frame))

		imgui.TableNextColumn()
		imgui.Text(fmt.Sprintf("Scanline: %d", coords.Scanline))

		imgui.TableNextColumn()
		imgui.Text(fmt.Sprintf("Clock: %d", coords.Clock))

		imgui.TableNextColumn()
		signal := fmt.Sprintf("%s", win.img.cache.TV.GetLastSignal().String())
		if !win.scr.crit.frameInfo.FromVSYNC {
			signal = fmt.Sprintf("%sUNSYNCED", signal)
		}
		imgui.Text(signal)

		imgui.EndTable()
	}
}

func (win *winDbgScr) drawOverlayCombo() {
	coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
	imgui.PushItemWidth(win.overlayComboDim.X + imgui.FrameHeight())

	// change coprocessor text to CoProcID if a coprocessor is present
	v := win.img.screen.crit.overlay
	if v == reflection.OverlayLabels[reflection.OverlayCoproc] {
		if coproc != nil {
			v = coproc.ProcessorID()
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
			} else if coproc != nil {
				// if ROM has a coprocessor change the option label to the
				// appropriate coprocessor ID
				if imgui.Selectable(coproc.ProcessorID()) {
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

func (win *winDbgScr) drawOverlayComboTooltip() {
	switch win.img.screen.crit.overlay {
	case reflection.OverlayLabels[reflection.OverlayVBLANK_VSYNC]:
		win.img.imguiTooltip(func() {
			imguiColorLabelSimple("VBLANK", win.img.cols.reflectionColors[reflection.VBLANK])
			imgui.Spacing()
			imguiColorLabelSimple("VSYNC", win.img.cols.reflectionColors[reflection.VSYNC_WITH_VBLANK])
			imgui.Spacing()
			imguiColorLabelSimple("VSYNC without VBLANK", win.img.cols.reflectionColors[reflection.VSYNC_NO_VBLANK])
		}, true)
	case reflection.OverlayLabels[reflection.OverlayWSYNC]:
		win.img.imguiTooltip(func() {
			imguiColorLabelSimple("WSYNC", win.img.cols.reflectionColors[reflection.WSYNC])
		}, true)
	case reflection.OverlayLabels[reflection.OverlayWSYNC]:
		win.img.imguiTooltip(func() {
			imguiColorLabelSimple("WSYNC", win.img.cols.reflectionColors[reflection.WSYNC])
		}, true)
	case reflection.OverlayLabels[reflection.OverlayCollision]:
		win.img.imguiTooltip(func() {
			imguiColorLabelSimple("Collision", win.img.cols.reflectionColors[reflection.Collision])
			imgui.Spacing()
			imguiColorLabelSimple("CXCLR", win.img.cols.reflectionColors[reflection.CXCLR])
		}, true)
	case reflection.OverlayLabels[reflection.OverlayHMOVE]:
		win.img.imguiTooltip(func() {
			imguiColorLabelSimple("Delay", win.img.cols.reflectionColors[reflection.HMOVEdelay])
			imgui.Spacing()
			imguiColorLabelSimple("Ripple", win.img.cols.reflectionColors[reflection.HMOVEripple])
			imgui.Spacing()
			imguiColorLabelSimple("Latch", win.img.cols.reflectionColors[reflection.HMOVElatched])
		}, true)
	case reflection.OverlayLabels[reflection.OverlayRSYNC]:
		win.img.imguiTooltip(func() {
			imguiColorLabelSimple("Align", win.img.cols.reflectionColors[reflection.RSYNCalign])
			imgui.Spacing()
			imguiColorLabelSimple("Reset", win.img.cols.reflectionColors[reflection.RSYNCreset])
		}, true)
	case reflection.OverlayLabels[reflection.OverlayCoproc]:
		win.img.imguiTooltip(func() {
			coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
			if coproc == nil {
				imgui.Text("no coprocessor")
			} else {
				key := fmt.Sprintf("parallel %s", coproc.ProcessorID())
				imguiColorLabelSimple(key, win.img.cols.reflectionColors[reflection.CoProcActive])
			}
		}, true)
	}
}

// called from within a win.scr.crit.section Lock().
func (win *winDbgScr) drawReflectionTooltip() {
	if win.isCaptured || !win.mouse.valid {
		return
	}

	// no useful reflection if mouse is outside TV area
	if win.mouse.tv.Scanline > win.scr.crit.frameInfo.TotalScanlines {
		imguiTooltipSimple("no TV signal", true)
		return
	}

	// get reflection information
	ref := win.scr.crit.reflection[win.mouse.offset]

	e := win.img.dbg.Disasm.FormatResult(ref.Bank, ref.CPU, disassembly.EntryLevelBlessed)

	// the magnify tooltip needs to appear before anything else and we only
	// want to draw it if there is no overlay and there is an instruction
	// behind the pixel
	if e.Address != "" && win.scr.crit.overlay == reflection.OverlayLabels[reflection.OverlayNone] {
		// we also want to show it regardless of the global tooltip preference
		// if the magnify show tooltip field is true
		imguiTooltip(func() {
			win.magnifyTooltip.draw(win.mouse)
		}, false, win.magnifyTooltip.showInTooltip)
	}

	// draw tooltip
	win.img.imguiTooltip(func() {
		// separator if we've drawn the magnification
		if win.magnifyTooltip.showInTooltip {
			imguiSeparator()
		}

		imgui.Text(fmt.Sprintf("Scanline: %d", win.mouse.tv.Scanline))
		imgui.Text(fmt.Sprintf("Clock: %d", win.mouse.tv.Clock))

		// early return if there is no instruction behind this pixel
		if e.Address == "" {
			return
		}

		imguiSeparator()

		// if mouse is over a pixel from the previous frame then show a note
		// if win.img.dbg.State() == govern.Paused {
		// 	if win.mouse.tv.Scanline > win.img.screen.crit.lastScanline ||
		// 		(win.mouse.tv.Scanline == win.img.screen.crit.lastScanline && win.mouse.tv.Clock > win.img.screen.crit.lastClock) {
		// 		imgui.Text("From previous frame")
		// 		imguiSeparator()
		// 	}
		// }

		// pixel swatch. using black swatch if pixel is HBLANKed or VBLANKed
		var px signal.ColorSignal
		var label string
		if (ref.IsHblank || ref.Signal.VBlank || px == signal.VideoBlack) && !win.elements {
			px = signal.VideoBlack
			label = "No color signal"
		} else {
			px = ref.Signal.Color
			label = ref.VideoElement.String()
		}

		spec := win.img.cache.TV.GetFrameInfo().Spec
		rgba := spec.GetColor(px)
		col := imgui.Vec4{
			X: float32(rgba.R) / 255, Y: float32(rgba.G) / 255, Z: float32(rgba.B) / 255, W: float32(rgba.A) / 255,
		}
		imgui.PushStyleColor(imgui.StyleColorText, col)
		imgui.Text(string(fonts.ColorSwatch))
		imgui.PopStyleColor()
		imgui.SameLine()
		imgui.Text(label)

		// instruction information
		imgui.Spacing()
		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmBreakAddress)
		if win.img.cache.VCS.Mem.Cart.NumBanks() > 1 {
			imgui.Text(fmt.Sprintf("%s [bank %s]", e.Address, ref.Bank))
		} else {
			imgui.Text(e.Address)
		}
		imgui.PopStyleColor()

		imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperator)
		imgui.Text(e.Operator)
		imgui.PopStyleColor()

		if e.Operand.Resolve() != "" {
			imgui.SameLine()
			imgui.PushStyleColor(imgui.StyleColorText, win.img.cols.DisasmOperand)
			imgui.Text(e.Operand.Resolve())
			imgui.PopStyleColor()
		}

		switch win.scr.crit.overlay {
		case reflection.OverlayLabels[reflection.OverlayNone]:
			fallthrough

		case reflection.OverlayLabels[reflection.OverlayVBLANK_VSYNC]:
			// tooltip for VBLANK/VSYNC overlay is the same as for when there is
			// no overlay
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

			// TODO: these bits are the current bits and not the bits under the cursor

			cxm0p, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXM0P])
			cxm1p, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXM1P])
			cxp0fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXP0FB])
			cxp1fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXP1FB])
			cxm0fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXM0FB])
			cxm1fb, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXM1FB])
			cxblpf, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXBLPF])
			cxppmm, _ := win.img.cache.VCS.Mem.Peek(cpubus.ReadAddressByRegister[cpubus.CXPPMM])

			imguiLabel("CXM0P ")
			drawRegister("##CXM0P", cxm0p, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXM1P ")
			drawRegister("##CXM1P", cxm1p, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXP0FB")
			drawRegister("##CXP0FB", cxp0fb, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXP1FB")
			drawRegister("##CXP1FB", cxp1fb, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXM0FB")
			drawRegister("##CXM0FB", cxm0fb, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXM1FB")
			drawRegister("##CXM1FB", cxm1fb, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXBLPF")
			drawRegister("##CXBLPF", cxblpf, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)
			imguiLabel("CXPPMM")
			drawRegister("##CXPPMM", cxppmm, vcs.TIADrivenPins, win.img.cols.collisionBit, nil)

			imguiSeparator()

			s := ref.Collision.LastColorClock.String()
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
			coproc := win.img.cache.VCS.Mem.Cart.GetCoProc()
			if coproc == nil {
				imgui.Text("no coprocessor")
			} else {
				id := coproc.ProcessorID()
				imguiSeparator()
				switch ref.CoProcSync {
				case coprocessor.CoProcIdle:
					imgui.Text(fmt.Sprintf("%s is idle", id))
				case coprocessor.CoProcNOPFeed:
					imgui.Text(fmt.Sprintf("%s is feeding NOPs", id))
				case coprocessor.CoProcStrongARMFeed:
					imgui.Text(fmt.Sprintf("%s feeding 6507", id))
				case coprocessor.CoProcParallel:
					imgui.Text(fmt.Sprintf("%s and 6507 running in parallel", id))
				}
			}
		}
	}, false)
}

// resize() implements the textureRenderer interface.
func (win *winDbgScr) resize() {
	win.displayTexture.markForCreation()
	win.elementsTexture.markForCreation()
	win.overlayTexture.markForCreation()

	// scaling is set every render() rather than on resize(), like it is done
	// in the playscr. this is because scaling needs to consider the size of
	// the dbgscr window which can change more often than resize() is called.
}

// updateRefreshRate() implements the textureRenderer interface.
func (win *winDbgScr) updateRefreshRate() {
}

// render() implements the textureRenderer interface.
//
// render is called by service loop (via screen.render()). must be inside
// screen critical section.
func (win *winDbgScr) render() {
	if win.cropped {
		win.displayTexture.render(win.scr.crit.cropPixels)
		win.elementsTexture.render(win.scr.crit.cropElementPixels)
		win.overlayTexture.render(win.scr.crit.cropOverlayPixels)
	} else {
		win.displayTexture.render(win.scr.crit.presentationPixels)
		win.elementsTexture.render(win.scr.crit.elementPixels)
		win.overlayTexture.render(win.scr.crit.overlayPixels)
	}

	if win.magnifyTooltip.clip.Size().X > 0 {
		var src *image.RGBA
		if win.elements {
			src = win.scr.crit.elementPixels
		} else {
			src = win.scr.crit.presentationPixels
		}

		pixels := src.SubImage(win.magnifyTooltip.clip).(*image.RGBA)
		if !pixels.Rect.Size().Eq(win.magnifyTooltip.clip.Size()) {
			if win.magnifyTooltip.clip.Min.X < 0 {
				win.magnifyTooltip.clip.Max.X -= win.magnifyTooltip.clip.Min.X
				win.magnifyTooltip.clip.Min.X = 0
			} else if win.magnifyTooltip.clip.Max.X > pixels.Rect.Max.X {
				d := win.magnifyTooltip.clip.Max.X - pixels.Rect.Max.X
				win.magnifyTooltip.clip.Max.X -= d
				win.magnifyTooltip.clip.Min.X -= d
			}
			if win.magnifyTooltip.clip.Min.Y < 0 {
				win.magnifyTooltip.clip.Max.Y -= win.magnifyTooltip.clip.Min.Y
				win.magnifyTooltip.clip.Min.Y = 0
			} else if win.magnifyTooltip.clip.Max.Y > pixels.Rect.Max.Y {
				d := win.magnifyTooltip.clip.Max.Y - pixels.Rect.Max.Y
				win.magnifyTooltip.clip.Max.Y -= d
				win.magnifyTooltip.clip.Min.Y -= d
			}
			pixels = src.SubImage(win.magnifyTooltip.clip).(*image.RGBA)
		}

		win.magnifyTooltip.texture.markForCreation()
		win.magnifyTooltip.texture.render(pixels)
	}

	if win.magnifyWindow.clip.Size().X > 0 {
		var src *image.RGBA
		if win.elements {
			src = win.scr.crit.elementPixels
		} else {
			src = win.scr.crit.presentationPixels
		}

		pixels := src.SubImage(win.magnifyWindow.clip).(*image.RGBA)
		if !pixels.Rect.Size().Eq(win.magnifyWindow.clip.Size()) {
			if win.magnifyWindow.clip.Min.X < 0 {
				win.magnifyWindow.clip.Max.X -= win.magnifyWindow.clip.Min.X
				win.magnifyWindow.clip.Min.X = 0
				win.magnifyWindow.centerPoint.x = win.magnifyWindow.clip.Max.X / 2
			} else if win.magnifyWindow.clip.Max.X > pixels.Rect.Max.X {
				d := win.magnifyWindow.clip.Max.X - pixels.Rect.Max.X
				win.magnifyWindow.clip.Max.X -= d
				win.magnifyWindow.clip.Min.X -= d
				win.magnifyWindow.centerPoint.x -= d
			}
			if win.magnifyWindow.clip.Min.Y < 0 {
				win.magnifyWindow.clip.Max.Y -= win.magnifyWindow.clip.Min.Y
				win.magnifyWindow.clip.Min.Y = 0
				win.magnifyWindow.centerPoint.y = win.magnifyWindow.clip.Max.Y / 2
			} else if win.magnifyWindow.clip.Max.Y > pixels.Rect.Max.Y {
				d := win.magnifyWindow.clip.Max.Y - pixels.Rect.Max.Y
				win.magnifyWindow.clip.Max.Y -= d
				win.magnifyWindow.clip.Min.Y -= d
				win.magnifyWindow.centerPoint.y -= d
			}
			pixels = src.SubImage(win.magnifyWindow.clip).(*image.RGBA)
		}

		win.magnifyWindow.texture.markForCreation()
		win.magnifyWindow.texture.render(pixels)
	}
}

// must be called from with a critical section.
func (win *winDbgScr) setScaling() {

	// aspectBias transforms the scaling factor for the X axis. in other words,
	// for width of every pixel is height of every pixel multiplied by the
	// aspect bias
	const aspectBias = 0.91

	var w float32
	var h float32

	if win.cropped {
		w = float32(win.scr.crit.cropPixels.Bounds().Size().X)
		h = float32(win.scr.crit.cropPixels.Bounds().Size().Y)
	} else {
		w = float32(win.scr.crit.presentationPixels.Bounds().Size().X)
		h = float32(win.scr.crit.presentationPixels.Bounds().Size().Y)
	}
	adjW := w * pixelWidth * float32(aspectBias)

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
	win.xscaling = scaling * pixelWidth * float32(aspectBias)
	win.scaledWidth = w * win.xscaling
	win.scaledHeight = h * win.yscaling

	// get numscanlines while we're in critical section
	win.numScanlines = win.scr.crit.frameInfo.VisibleBottom - win.scr.crit.frameInfo.VisibleTop
}
