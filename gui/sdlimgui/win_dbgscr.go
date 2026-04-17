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

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/gui/fonts"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/memory/vcs"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/video"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/imgui-go/v5"
)

const (
	winDbgScrID        = "TV Screen"
	winDbgScrMagnifyID = "TV Screen Magnify"
	winDbgScrTooltipID = "##tvscreentooltip"
)

type winDbgScrMode int

const (
	winDbgScrNormal winDbgScrMode = iota
	winDbgScrMagnify
	winDbgScrTooltip
)

type winDbgScrView struct {
	mode winDbgScrMode

	// origin (position) of image on the screen
	screenOrigin imgui.Vec2

	// scaling of texture and calculated dimensions
	xscaling     float32
	yscaling     float32
	scaledWidth  float32
	scaledHeight float32

	// whether the HBLANK/VBLANK areas are cropped
	cropped bool
}

type winDbgScr struct {
	debuggerWin

	img *SdlImgui

	// reference to screen data
	scr *screen

	// view is required by the dbgscr shader
	view winDbgScrView

	// textures
	displayTeture   texture
	elementsTexture texture
	overlayTexture  texture

	// whether to use debug colours for the screen
	elements bool

	// the tv screen has captured mouse input
	isCaptured bool

	// the current position of the actual mouse (not reliable after the frame it
	// was captured on)
	mouse dbgScrMouse

	// whether the mouse buttom (numbered 0=left, 1=right, 2=middle) is being held from last frame
	mouseDragging [3]bool

	// height of tool bar at bottom of window. valid after first frame.
	toolbarHeight float32

	// additional padding for the image so that it is centred in its content space
	imagePadding imgui.Vec2

	// size of area available to the screen image
	screenRegion imgui.Vec2

	// the dimensions required for the combo widgets
	specComboDim    imgui.Vec2
	overlayComboDim imgui.Vec2

	// number of scanlines in current image. taken from screen but is crit section safe
	numScanlines int

	// magnification fields
	magnifyWindow  *winDbgScr
	magnifyTooltip *winDbgScr

	// whether to show the magnify tooltip
	showMagnifyInTooltip bool
}

func newWinDbgScr(img *SdlImgui, mode winDbgScrMode) (*winDbgScr, error) {
	win := &winDbgScr{
		img: img,
		scr: img.screen,
		view: winDbgScrView{
			mode:    mode,
			cropped: true,
		},
	}
	win.debuggerGeom.noFocusTracking = true

	// set texture, creation of textures will be done after every call to resize()
	win.displayTeture = img.rnd.addTexture(shaderDbgScr, true, true, &win.view)
	win.elementsTexture = img.rnd.addTexture(shaderDbgScr, true, true, &win.view)
	win.overlayTexture = img.rnd.addTexture(shaderDbgScrOverlay, false, false, &win.view)

	// call setScaling() now so that render() has something to work with - even
	// though setScaling() is called every draw if the window is open it will
	// leave render() nothing to work with if it isn't open on startup
	win.setScaling()

	// create magnify window if this is a normal mode winDbgScr
	if win.view.mode == winDbgScrNormal {
		var err error
		win.magnifyWindow, err = newWinDbgScr(img, winDbgScrMagnify)
		if err != nil {
			return nil, err
		}
	}

	// create magnify tooltip for all modes except a tooltip
	if win.view.mode != winDbgScrTooltip {
		var err error
		win.magnifyTooltip, err = newWinDbgScr(img, winDbgScrTooltip)
		if err != nil {
			return nil, err
		}
	}

	return win, nil
}

func (win *winDbgScr) init() {
	win.specComboDim = imguiGetFrameDim("", specification.SpecList...)
	win.overlayComboDim = imguiGetFrameDim("", reflection.OverlayLabels...)
	if win.magnifyWindow != nil {
		win.magnifyWindow.init()
	}
	if win.magnifyTooltip != nil {
		win.magnifyTooltip.init()
	}
}

func (win *winDbgScr) id() string {
	switch win.view.mode {
	case winDbgScrNormal:
		return winDbgScrID
	case winDbgScrMagnify:
		return winDbgScrMagnifyID
	case winDbgScrTooltip:
		return winDbgScrTooltipID
	}
	panic("unknown winDbgScr mode")
}

const contextMenu = "dbgScreenContextMenu"

func (win *winDbgScr) debuggerDraw() bool {
	// if window isn't open then child windows are not drawn either
	if !win.debuggerIsOpen() {
		return false
	}

	if win.view.mode == winDbgScrTooltip {
		return win.drawView()
	}

	if win.magnifyWindow != nil {
		_ = win.magnifyWindow.debuggerDraw()
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
		// note size of remaining window and content area
		win.screenRegion = imgui.ContentRegionAvail()
		win.screenRegion.Y -= win.toolbarHeight

		// screen image, overlays, menus and tooltips
		imgui.BeginChildV("##image", imgui.Vec2{X: win.screenRegion.X, Y: win.screenRegion.Y}, false, imgui.ChildFlagsNone)

		// add horiz/vert padding around screen image
		imgui.SetCursorPos(imgui.CursorPos().Plus(win.imagePadding))

		// draw main view of winDbgScr. whether the mouse is hovered over the view is returned
		imageHovered := win.drawView()

		// support for paintbox tool
		win.paintDragAndDrop()

		// get mouse position if context menu is not open
		if !imgui.IsPopupOpen(contextMenu) {
			win.mouse = currentDbgScrMouse(win.scr, win.view)
		}

		// popup menu on right mouse button
		//
		// we only call OpenPopup() if it's not already open. also, care taken to
		// avoid menu opening when releasing a captured mouse.
		if !win.isCaptured && (win.mouseDragging[1] || (imageHovered && imgui.IsMouseClicked(1))) {
			win.mouseDragging[1] = imgui.IsMouseDown(1)
			imgui.OpenPopup(contextMenu)
		}

		if imgui.BeginPopup(contextMenu) {
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
			if win.view.mode == winDbgScrNormal {
				imguiSeparator()
				if imgui.Selectable(fmt.Sprintf("%c Magnify in Window", fonts.MagnifyingGlass)) {
					if win.magnifyWindow != nil {
						win.magnifyWindow.debuggerSetOpen(true)
						// win.magnifyWindow.setClipCenter(win.mouse, win.scr.crit.presentationPixels.Bounds())
					}
				}
			}
			imgui.EndPopup()
		}

		// draw tool tip
		if imgui.IsItemHovered() {
			win.drawReflectionTooltip()
		}

		// left-mouse button will cause the rewind goto coords to run only when the emulation is paused
		if imgui.IsWindowFocused() && win.img.dbg.State() == govern.Paused {
			// handle dragging outside of display boundaries
			if win.mouseDragging[0] || (imageHovered && imgui.IsMouseClicked(0)) {
				win.mouseDragging[0] = imgui.IsMouseDown(0)

				current := win.img.cache.TV.GetCoords()
				to := coords.TelevisionCoords{
					Frame:    current.Frame,
					Scanline: win.mouse.tv.Scanline,
					Clock:    win.mouse.tv.Clock,
				}

				if !coords.Equal(current, to) {
					win.img.dbg.GotoCoords(to)
				}
			}
		}

		// move pivot point with middle mouse button. handle dragging outside of display boundaries
		if win.mouseDragging[2] || (imageHovered && imgui.IsMouseClicked(2)) {
			win.mouseDragging[2] = imgui.IsMouseDown(2)
			if win.magnifyWindow != nil {
				win.magnifyWindow.debuggerSetOpen(true)
				// win.magnifyWindow.setClipCenter(win.mouse, win.scr.crit.presentationPixels.Bounds())
			}
		}

		// end of screen image
		imgui.EndChild()

		// toolbar at bottom of window
		if win.view.mode == winDbgScrNormal {
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
				imgui.Text(fmt.Sprintf("%.1fx", win.view.yscaling))

				// debugging toggles
				imgui.SameLineV(0, 15)
				if imgui.Checkbox("Debug Colours", &win.elements) {
					if win.magnifyWindow != nil {
						win.magnifyWindow.elements = win.elements
					}
					if win.magnifyTooltip != nil {
						win.magnifyTooltip.elements = win.elements
					}
				}

				imgui.SameLineV(0, 15)
				if imgui.Checkbox("Cropping", &win.view.cropped) {
					if win.magnifyWindow != nil {
						win.magnifyWindow.view.cropped = win.view.cropped
					}
					if win.magnifyTooltip != nil {
						win.magnifyTooltip.view.cropped = win.view.cropped
					}
					win.resize()
				}

				imgui.SameLineV(0, 15)
				win.drawOverlayCombo()

				if win.img.screen.crit.overlay == reflection.OverlayLabels[reflection.OverlayNone] {
					imgui.SameLineV(0, 15)
					if imgui.Checkbox("Magnify on hover", &win.showMagnifyInTooltip) {
						win.magnifyTooltip.debuggerSetOpen(win.showMagnifyInTooltip)
					}
				}
			})
		}
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winDbgScr) drawView() bool {
	// note the current cursor position. we'll use this to everything to the
	// corner of the screen.
	win.view.screenOrigin = imgui.CursorScreenPos()

	// push style info for screen and overlay ImageButton(). we're using
	// ImageButton because an Image will not capture mouse events and pass them
	// to the parent window. this means that a click-drag on the screen/overlay
	// will move the window, which we don't want.
	imgui.PushStyleColor(imgui.StyleColorButton, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonActive, win.img.cols.Transparent)
	imgui.PushStyleColor(imgui.StyleColorButtonHovered, win.img.cols.Transparent)
	imgui.PushStyleVarVec2(imgui.StyleVarFramePadding, imgui.Vec2{X: 0.0, Y: 0.0})

	defer imgui.PopStyleVar()
	defer imgui.PopStyleColorV(3)

	var textureID uint32
	if win.elements {
		textureID = win.elementsTexture.getID()
	} else {
		textureID = win.displayTeture.getID()
	}

	imgui.ImageButton(fmt.Sprintf("%sdisplay", win.id()), imgui.TextureID(textureID), imgui.Vec2{X: win.view.scaledWidth, Y: win.view.scaledHeight})
	hover := imgui.IsItemHovered()

	// overlay texture on top of screen texture
	imgui.SetCursorScreenPos(win.view.screenOrigin)
	imgui.ImageButton(fmt.Sprintf("%soverlay", win.id()), imgui.TextureID(win.overlayTexture.getID()), imgui.Vec2{X: win.view.scaledWidth, Y: win.view.scaledHeight})

	return hover
}

func (win *winDbgScr) drawSpecCombo() {
	spec := win.img.cache.TV.GetFrameInfo().Spec.ID

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
		if imgui.Selectable("Auto") {
			win.img.term.pushCommand("TV SPEC AUTO")
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

	// draw tooltip for overlay combo
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
		win.img.imguiTooltipSimple("no TV signal")
		return
	}

	// get reflection information
	ref := win.scr.crit.reflection[win.mouse.offset]

	e := win.img.dbg.Disasm.FormatResultAdHoc(ref.Bank, ref.CPU)

	win.img.imguiTooltip(func() {
		imgui.Text(fmt.Sprintf("Scanline: %d", win.mouse.tv.Scanline))
		imgui.Text(fmt.Sprintf("Clock: %d", win.mouse.tv.Clock))

		// early return if there is no instruction behind this pixel
		if e.Address == "" {
			return
		}

		imguiSeparator()

		spec := win.img.cache.TV.GetFrameInfo().Spec

		// pixel swatch. using black swatch if pixel is HBLANKed or VBLANKed
		var px signal.ColorSignal
		if (ref.IsHblank || ref.Signal.VBlank || px == signal.ZeroBlack) && !win.elements {
			imguiColorLabelSimple(
				"No color signal",
				colorRGBAtoVec4(spec.GetColor(signal.ZeroBlack)),
			)
		} else {
			switch ref.VideoElement {
			case video.ElementPlayfield:
				imguiColorLabelSimple(
					fmt.Sprintf("%s [PF%d]", ref.VideoElement.String(), ref.VideoElementCt),
					colorRGBAtoVec4(spec.GetColor(ref.Signal.Color)),
				)
			case video.ElementPlayer0, video.ElementPlayer1, video.ElementMissile0, video.ElementMissile1:
				var tag string
				switch ref.VideoElementCt {
				case 0:
					tag = " [1st copy]"
				case 1:
					tag = " [2nd copy]"
				case 2:
					tag = " [3rd copy]"
				}
				imguiColorLabelSimple(
					fmt.Sprintf("%s%s", ref.VideoElement.String(), tag),
					colorRGBAtoVec4(spec.GetColor(ref.Signal.Color)),
				)
			default:
				imguiColorLabelSimple(
					ref.VideoElement.String(),
					colorRGBAtoVec4(spec.GetColor(ref.Signal.Color)),
				)
			}
		}

		imgui.SameLine()

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
			if ref.Hmove.Future.IsActive() {
				imgui.Text(fmt.Sprintf("HMOVE delay: %d", ref.Hmove.FutureLatch.Remaining()))
			} else if ref.Hmove.Latch {
				if ref.Hmove.Ripple != 0xff {
					imgui.Text(fmt.Sprintf("HMOVE ripple: %d", ref.Hmove.FutureLatch.Remaining()))
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

		// update pxe colours window with information about location of mouse pointer over the debug
		// screen window. whether the arrow is acutally drawn is controlled by the colours window
		win.img.wm.windows[winPXEColoursID].(*winPXEColours).clearArrow()
		if sref, ok := win.img.screen.findReflection(ref, win.mouse.offset); ok {
			win.img.wm.windows[winPXEColoursID].(*winPXEColours).setArrow(sref.PXEPaletteAddr)
		}
	}, false)

	// the magnify tooltip needs to appear before anything else and we only
	// want to draw it if there is no overlay and there is an instruction
	// behind the pixel
	if e.Address != "" && win.scr.crit.overlay == reflection.OverlayLabels[reflection.OverlayNone] {
		// we also want to show it regardless of the global tooltip preference
		// if the magnify show tooltip field is true
		imguiTooltip(func() {
			imguiSeparator()
			win.magnifyTooltip.debuggerDraw()
		}, false, win.showMagnifyInTooltip)
	}
}

// resize() implements the textureRenderer interface.
func (win *winDbgScr) resize() {
	if win.magnifyWindow != nil {
		win.magnifyWindow.resize()
	}

	if win.magnifyTooltip != nil {
		win.magnifyTooltip.resize()
	}

	win.displayTeture.markForCreation()
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
	if win.magnifyWindow != nil {
		win.magnifyWindow.render()
	}

	if win.magnifyTooltip != nil {
		win.magnifyTooltip.render()
	}

	if win.view.cropped {
		win.displayTeture.render(win.scr.crit.cropPixels)
		win.elementsTexture.render(win.scr.crit.cropElementPixels)
		win.overlayTexture.render(win.scr.crit.cropOverlayPixels)
	} else {
		win.displayTeture.render(win.scr.crit.presentationPixels)
		win.elementsTexture.render(win.scr.crit.elementPixels)
		win.overlayTexture.render(win.scr.crit.overlayPixels)
	}
}

// must be called from with a critical section.
func (win *winDbgScr) setScaling() {
	if win.magnifyWindow != nil {
		win.magnifyWindow.setScaling()
	}

	if win.magnifyTooltip != nil {
		win.magnifyTooltip.setScaling()
	}

	// aspectBias transforms the scaling factor for the X axis. in other words,
	// for width of every pixel is height of every pixel multiplied by the
	// aspect bias
	const aspectBias = 0.91

	var w float32
	var h float32

	if win.view.cropped {
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

	win.view.yscaling = scaling
	win.view.xscaling = scaling * pixelWidth * float32(aspectBias)
	win.view.scaledWidth = w * win.view.xscaling
	win.view.scaledHeight = h * win.view.yscaling

	// get numscanlines while we're in critical section
	win.numScanlines = win.scr.crit.frameInfo.VisibleBottom - win.scr.crit.frameInfo.VisibleTop
}
