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
	img *SdlImgui
}

func newWinPXESymbols(img *SdlImgui) (window, error) {
	win := &winPXESymbols{img: img}
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
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPXESymbols) draw() {
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

	commit := func(address uint32, data uint8) {
		win.img.dbg.PushFunction(func() {
			win.img.dbg.VCS().Mem.Cart.GetStaticBus().ReferenceStatic().Write8bit(address, data)
		})
	}

	// there's no good way of determining whether there are any PXE symbols to display so we need to
	// be smart about how we call imgui.BeginTable()
	//
	// on the first iteration of the PXESymbols loop we call beginTable() and noting that it was
	// sucessful with the usingTable boolean. then at the end of the function we can either call
	// imgui.EndTable() or display the 'no symbols' text

	beginTable := func() bool {
		flgs := imgui.TableFlagsBordersInnerV |
			imgui.TableFlagsScrollY |
			imgui.TableFlagsSizingStretchProp |
			imgui.TableFlagsResizable

		if imgui.BeginTableV("##pxesymbols", 3, flgs, imgui.Vec2{}, 0) {
			width := imgui.ContentRegionAvail().X

			imgui.TableSetupColumnV("Symbol", imgui.TableColumnFlagsPreferSortDescending, width*0.45, 0)
			imgui.TableSetupColumnV("Address", imgui.TableColumnFlagsPreferSortDescending, width*0.25, 0)
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

		imgui.TableNextRow()
		if imgui.TableNextColumn() {
			imgui.Text(e.Symbol)
		}

		address := origin + uint32(e.Address)

		if imgui.TableNextColumn() {
			imgui.Textf("%08x\n", address)
		}

		if imgui.TableNextColumn() {
			v, ok := mem.Read8bit(address)
			if !ok {
				imgui.Text("illegal address")
			} else {
				s := fmt.Sprintf("%02x", uint8(v))
				if imguiHexInput(fmt.Sprintf("##pxe%8x", address), 2, &s) {
					win.img.dbg.PushFunction(func() {
						if v, err := strconv.ParseUint(s, 16, 8); err == nil {
							commit(address, uint8(v))
						}
					})
				}
			}
		}
	}

}
