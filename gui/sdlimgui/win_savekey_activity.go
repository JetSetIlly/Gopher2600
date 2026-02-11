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
	"strconv"

	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/imgui-go/v5"
)

const winSaveKeyActivityID = "SaveKey Activity"
const winSaveKeyActivityMenu = "Activity"

type winSaveKeyActivity struct {
	debuggerWin

	img *SdlImgui

	// savekey instance
	savekey *savekey.SaveKey
}

func newWinSaveKeyActivity(img *SdlImgui) (window, error) {
	win := &winSaveKeyActivity{
		img: img,
	}

	return win, nil
}

func (win *winSaveKeyActivity) init() {
}

func (win *winSaveKeyActivity) id() string {
	return winSaveKeyActivityID
}

func (win *winSaveKeyActivity) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not draw if savekey is not active
	win.savekey = win.img.cache.VCS.GetSaveKey()
	if win.savekey == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 633, Y: 358}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsAlwaysAutoResize) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winSaveKeyActivity) draw() {
	win.drawAddress()
	imgui.SameLine()
	win.drawBits()
	imgui.SameLine()
	win.drawACK()

	imgui.Spacing()
	style := imgui.CurrentStyle()
	dim := imgui.Vec2{
		X: imgui.WindowWidth() - ((style.FramePadding().X * 2) + (style.ItemInnerSpacing().X * 2)),
		Y: imgui.FrameHeight() * 2}
	drawI2C(win.savekey.SCL, win.savekey.SDA, dim, win.img.cols, win.img)

	imgui.Spacing()
	win.drawStatus()
}

func (win *winSaveKeyActivity) drawStatus() {
	imgui.AlignTextToFramePadding()
	switch win.savekey.State {
	case savekey.SaveKeyStopped:
		imgui.Text("Stopped")
	case savekey.SaveKeyStarting:
		imgui.Text("Starting")
	case savekey.SaveKeyAddressHi:
		fallthrough
	case savekey.SaveKeyAddressLo:
		imgui.Text("Getting address")
	case savekey.SaveKeyData:
		switch win.savekey.Dir {
		case savekey.Reading:
			imgui.Text("Reading")
		case savekey.Writing:
			imgui.Text("Writing")
		}
		imgui.SameLine()
		imgui.Text("Data")
	}
}

func (win *winSaveKeyActivity) drawACK() {
	v := win.savekey.Ack
	imgui.AlignTextToFramePadding()
	imgui.Text("ACK")
	imgui.SameLine()
	if imgui.Checkbox("##savekeyACK", &v) {
		win.img.dbg.PushFunction(func() {
			if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
				sk.Ack = v
			} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
				vox.SaveKey.Ack = v
			}
		})
	}
}

func (win *winSaveKeyActivity) drawBits() {
	bits := win.savekey.Bits
	bitCt := win.savekey.BitsCt

	var label string
	switch win.savekey.Dir {
	case savekey.Reading:
		label = "Reading"
	case savekey.Writing:
		label = "Writing"
	}

	s := fmt.Sprintf("%02x", bits)
	imguiLabel(label)
	if imguiHexInput(fmt.Sprintf("##savekey%s", label), 2, &s) {
		v, err := strconv.ParseUint(s, 16, 8)
		if err != nil {
			panic(err)
		}
		win.img.dbg.PushFunction(func() {
			if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
				sk.Bits = uint8(v)
			} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
				vox.SaveKey.Bits = uint8(v)
			}
		})
	}

	imgui.SameLine()

	seq := newDrawlistSequence(imgui.Vec2{X: imgui.FrameHeight() * 0.75, Y: imgui.FrameHeight() * 0.75}, true)
	for i := range 8 {
		if (bits<<i)&0x80 != 0x80 {
			seq.nextItemDepressed = true
		}
		if seq.rectFill(win.img.cols.saveKeyBit) {
			v := bits ^ (0x80 >> i)
			win.img.dbg.PushFunction(func() {
				if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
					sk.Bits = v
				} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
					vox.SaveKey.Bits = v
				}
			})
		}
		seq.sameLine()
	}
	seq.end()

	if win.savekey.State != savekey.SaveKeyStopped && bitCt < 8 {
		dl := imgui.WindowDrawList()
		dl.AddCircleFilled(imgui.Vec2{X: seq.offsetX(bitCt), Y: imgui.CursorScreenPos().Y},
			imgui.FontSize()*0.20, win.img.cols.saveKeyBitPointer)
	}
}

func (win *winSaveKeyActivity) drawAddress() {
	addr := win.savekey.EEPROM.Address

	label := "Address"
	s := fmt.Sprintf("%04x", addr)
	imguiLabel(label)
	if imguiHexInput(fmt.Sprintf("##savekey%s", label), 4, &s) {
		v, err := strconv.ParseUint(s, 16, 16)
		if err != nil {
			panic(err)
		}
		win.img.dbg.PushFunction(func() {
			if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
				sk.EEPROM.Address = uint16(v)
			} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
				vox.SaveKey.EEPROM.Address = uint16(v)
			}
		})
	}
}
