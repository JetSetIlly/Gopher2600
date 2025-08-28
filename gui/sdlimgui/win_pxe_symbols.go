package sdlimgui

import (
	"fmt"
	"strconv"

	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/imgui-go/v5"
)

const winPXESymbolsID = "PXE Symbols"

type winPXESymbols struct {
	debuggerWin
	img          *SdlImgui
	popupPalette *popupPalette

	optionColourOnly bool
	optionsHeight    float32
}

func newWinPXESymbols(img *SdlImgui) (window, error) {
	win := &winPXESymbols{
		img:          img,
		popupPalette: newPopupPalette(img),
	}
	return win, nil
}

func (win *winPXESymbols) init() {
}

// id should return a unique identifier for the window. note that the
// window title and any menu entry do not have to have the same value as
// the id() but it can.
func (win *winPXESymbols) id() string {
	return winPXESymbolsID
}

func (win *winPXESymbols) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 978, Y: 164}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 551, Y: 589}, imgui.ConditionFirstUseEver)
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.drawSymbolTable()
		win.drawFilters()
		win.popupPalette.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPXESymbols) drawSymbolTable() {
	ef, ok := win.img.cache.VCS.Mem.Cart.GetCoProcBus().(coprocessor.CartCoProcELF)
	if !ok {
		imgui.Text("not an ELF cartridge")
		return
	}

	ok, origin := ef.PXE()
	if !ok {
		imgui.Text("not a PXE cartridge")
		return
	}

	bus, ok := ef.(mapper.CartStaticBus)
	if !ok {
		imgui.Text("PXE memory not initialised")
		return
	}
	mem := bus.GetStatic()

	_, ok = mem.Read8bit(origin)
	if !ok {
		imgui.Text("PXE memory not initialised")
		return
	}

	// there's no good way of determining whether there are any PXE symbols to display so we need to
	// be smart about how we call imgui.BeginTable()
	//
	// on the first iteration of the PXESymbols loop we call beginTable() and noting that it was
	// sucessful with the usingTable boolean. then at the end of the function we can either call
	// imgui.EndTable() or display the 'no symbols' text

	beginTable := func() bool {
		flgs := imgui.TableFlagsScrollY |
			imgui.TableFlagsSizingStretchSame |
			imgui.TableFlagsResizable

		sz := imgui.Vec2{
			Y: imguiRemainingWinHeight() - win.optionsHeight,
		}

		if imgui.BeginTableV("##pxesymbols", 3, flgs, sz, 0) {
			width := imgui.ContentRegionAvail().X

			imgui.TableSetupColumnV("Symbol", imgui.TableColumnFlagsPreferSortDescending, width*0.45, 0)
			if win.optionColourOnly {
				imgui.TableSetupColumnV("Palette Index", imgui.TableColumnFlagsPreferSortDescending, width*0.25, 0)
			} else {
				imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsPreferSortDescending, width*0.25, 0)
			}
			imgui.TableSetupColumnV("Value", imgui.TableColumnFlagsNoSort, width*0.25, 0)

			imgui.TableSetupScrollFreeze(0, 1)
			imgui.TableHeadersRow()
			return true
		}
		return false
	}

	var usingTable bool

	defer func() {
		if usingTable {
			imgui.EndTable()
		} else {
			imgui.Spacing()
			imgui.Text("No PXE symbols available")
		}
	}()

	for e := range win.img.dbg.Disasm.Sym.PXESymbols {
		if !usingTable {
			usingTable = beginTable()
			if !usingTable {
				break
			}
		}

		isColour := e.Address >= 0x0700 && e.Address <= 0x07ff

		if win.optionColourOnly && !isColour {
			continue
		}

		address := origin + uint32(e.Address)
		v, ok := mem.Read8bit(address)
		if !ok {
			// this shouldn't happen if the PXESymbols iterator is correct
			continue
		}

		imgui.TableNextRow()
		if imgui.TableNextColumn() {
			imgui.AlignTextToFramePadding()
			imgui.Text(e.Symbol)
		}

		if imgui.TableNextColumn() {
			imgui.AlignTextToFramePadding()
			if win.optionColourOnly {
				imgui.Textf("%02x\n", e.Address-0x0700)
			} else {
				imgui.Textf("%04x\n", e.Address)
			}
		}

		if imgui.TableNextColumn() {
			s := fmt.Sprintf("%02x", uint8(v))
			if imguiHexInput(fmt.Sprintf("##pxesymbol%s", e.Symbol), 2, &s) {
				win.img.dbg.PushFunction(func() {
					if v, err := strconv.ParseUint(s, 16, 8); err == nil {
						win.img.commitStaticMemory(address, uint8(v))
					}
				})
			}

			const (
				swatchSize    = 0.5
				swatchPadding = 10
			)

			imgui.SameLineV(0, swatchPadding)
			if isColour {
				v &= 0xfe
				if win.img.imguiTVColourSwatch(v, swatchSize) {
					win.popupPalette.request(&v, func() {
						win.img.commitStaticMemory(address, v)
					})
				}
			} else {
				imgui.Dummy(imgui.Vec2{X: imgui.FontSize()*swatchSize*2 + swatchPadding})
			}
		}
	}
}

func (win *winPXESymbols) drawFilters() {
	win.optionsHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Separator()
		imgui.Spacing()
		imgui.Checkbox("Colour symbols only##pxecolouronly", &win.optionColourOnly)
	})
}
