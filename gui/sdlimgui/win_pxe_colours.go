package sdlimgui

import (
	"github.com/jetsetilly/gopher2600/coprocessor"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/imgui-go/v5"
)

const winPXEColoursID = "PXE Colours"

type winPXEColours struct {
	debuggerWin
	img          *SdlImgui
	popupPalette *popupPalette
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

// id should return a unique identifier for the window. note that the
// window title and any menu entry do not have to have the same value as
// the id() but it can.
func (win *winPXEColours) id() string {
	return winPXEColoursID
}

func (win *winPXEColours) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 528, Y: 256}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
		win.popupPalette.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winPXEColours) draw() {
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

	for i := 0; i <= 0xff; i++ {
		address := origin + 0x0700 + uint32(i)
		v, ok := mem.Read8bit(address)
		v &= 0xfe
		if !ok {
			imgui.Text("illegal address")
		} else {
			if win.img.imguiTVColourSwatch(v, 0.75) {
				win.popupPalette.request(&v, func() {
					commit(address, v)
				})
			}
			if (i+1)%16 != 0 {
				imgui.SameLine()
			}
		}
	}
}
