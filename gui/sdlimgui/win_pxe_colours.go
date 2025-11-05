package sdlimgui

import (
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/elf"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/rewind"
	"github.com/jetsetilly/imgui-go/v5"
)

const winPXEColoursID = "PXE Colours"

type winPXEColours struct {
	debuggerWin
	img          *SdlImgui
	popupPalette *popupPalette

	// arrow control between the debug screen and the pxe colour palette
	arrowDraw    int
	arrowOrigin  imgui.Vec2
	arrowAddress uint32
	arrowShow    bool

	// PXE address for the colour at the current cursor position
	cursorAddress uint32
}

func newWinPXEColours(img *SdlImgui) (window, error) {
	win := &winPXEColours{
		img:          img,
		popupPalette: newPopupPalette(img),
	}
	return win, nil
}

func (win *winPXEColours) init() {
}

func (win *winPXEColours) id() string {
	return winPXEColoursID
}

func (win *winPXEColours) clearArrow() {
	win.arrowDraw = 0
}

func (win *winPXEColours) setArrow(address uint32) {
	win.arrowDraw = 2
	win.arrowOrigin = imgui.MousePos()
	win.arrowAddress = address
}

func (win *winPXEColours) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	ef, ok := win.img.cache.VCS.Mem.Cart.GetCoProcBus().(coprocessor.CartCoProcELF)
	if !ok {
		return false
	}

	ok, origin := ef.PXE()
	if !ok {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 528, Y: 256}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		if win.drawPalette(ef, origin) {
			win.popupPalette.draw()
			imguiSeparator()
			win.arrowShow = win.img.prefs.pxeColourIndicators.Get().(bool)
			if imgui.Checkbox("Show Indicators", &win.arrowShow) {
				win.img.prefs.pxeColourIndicators.Set(win.arrowShow)
			}
		}
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPXEColours) drawPalette(ef coprocessor.CartCoProcELF, origin uint32) bool {
	bus, ok := ef.(mapper.CartStaticBus)
	if !ok {
		imgui.Text("PXE memory not initialised")
		return false
	}
	mem := bus.GetStatic()

	_, ok = mem.Read8bit(origin)
	if !ok {
		imgui.Text("PXE memory not initialised")
		return false
	}

	for i := elf.PXEPaletteOrigin; i <= elf.PXEPaletteMemtop; i++ {
		address := origin + uint32(i)
		v, ok := mem.Read8bit(address)
		if ok {
			v &= 0xfe
			highlight := win.cursorAddress == address
			clicked, centre, r := win.img.imguiTVColourSwatchWithGeom(v, 0.75, highlight)
			if clicked {
				win.popupPalette.request(&v, func() {
					if win.img.dbg.State() != govern.Running {
						win.img.dbg.RerunLastNFrames(2, func(s *rewind.State) {
							s.VCS.Mem.Cart.GetStaticBus().ReferenceStatic().Write8bit(address, v)
						})
					}
				})
			}

			if (i+1)%16 != 0 {
				imgui.SameLine()
			}

			if win.arrowDraw > 0 && win.arrowShow {
				if address == win.arrowAddress {
					dl := imgui.ForegroundDrawList()
					dl.AddLineV(win.arrowOrigin, centre, win.img.cols.pxeColorArrow, 2)
					dl.AddCircleFilled(centre, r*0.35, win.img.cols.pxeColorArrow)
				}
			}
		}
	}

	return true
}

func (win *winPXEColours) postRender() {
	if win.arrowDraw > 0 {
		win.arrowDraw--
	}
}
