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

	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox"
	"github.com/jetsetilly/gopher2600/hardware/peripherals/savekey"
	"github.com/jetsetilly/imgui-go/v5"
)

const winSaveKeyEEPROMID = "SaveKey EEPROM"
const winSaveKeyEEPROMMenu = "EEPROM"

type winSaveKeyEEPROM struct {
	debuggerWin

	img *SdlImgui

	// height of status line at bottom of window. valid after first frame
	statusHeight float32

	// savekey instance
	savekey *savekey.SaveKey

	// scroll to scratchpad
	scrollScratch bool

	// whether to only show accessed pages
	showAccessedOnly bool
}

func newWinSaveKeyEEPROM(img *SdlImgui) (window, error) {
	win := &winSaveKeyEEPROM{img: img}
	return win, nil
}

func (win *winSaveKeyEEPROM) init() {
}

func (win *winSaveKeyEEPROM) id() string {
	return winSaveKeyEEPROMID
}

func (win *winSaveKeyEEPROM) debuggerDraw() bool {
	if !win.debuggerOpen {
		return false
	}

	// do not draw if savekey is not active
	win.savekey = win.img.cache.VCS.GetSaveKey()
	if win.savekey == nil {
		return false
	}

	imgui.SetNextWindowPosV(imgui.Vec2{X: 469, Y: 285}, imgui.ConditionFirstUseEver, imgui.Vec2{X: 0, Y: 0})
	imgui.SetNextWindowSizeV(imgui.Vec2{X: 478, Y: 356}, imgui.ConditionFirstUseEver)
	win.img.setReasonableWindowConstraints()

	if imgui.BeginV(win.debuggerID(win.id()), &win.debuggerOpen, imgui.WindowFlagsNone) {
		win.draw()
	}

	win.debuggerGeom.update()
	imgui.End()

	return true
}

func (win *winSaveKeyEEPROM) draw() {
	imgui.BeginChildV("eepromData", imgui.Vec2{X: 0, Y: imguiRemainingWinHeight() - win.statusHeight}, false, 0)

	var pagesShown bool

	for p := range savekey.EEPROMnumPages {
		if win.showAccessedOnly && !win.savekey.EEPROM.PageAccess[p] {
			continue
		}
		pagesShown = true

		origin := p * savekey.EEPROMpageSize
		memtop := origin + savekey.EEPROMpageSize - 1

		header := fmt.Sprintf("Page %03d (%04x - %04x)", p, origin, memtop)
		scratch := origin >= 0x3000 && origin < 0x4000
		if scratch {
			header = fmt.Sprintf("%s Scratchpad %d", header, ((origin-0x3000)/savekey.EEPROMpageSize)+1)
		}

		var flgs imgui.TreeNodeFlags
		if win.showAccessedOnly {
			flgs = imgui.TreeNodeFlagsDefaultOpen
		}
		drawByteGrid := imgui.CollapsingHeaderV(header, flgs)

		if scratch && win.scrollScratch {
			win.scrollScratch = false
			imgui.SetScrollHereY(0)
		}

		if drawByteGrid {
			d := win.savekey.EEPROM.Data[origin : memtop+1]
			dd := win.savekey.EEPROM.Disk[origin : memtop+1]
			win.img.drawByteGridSimple(fmt.Sprintf("eepromPage%d", p), d, dd, win.img.cols.ValueDiff, uint32(origin),
				func(idx int, data uint8) {
					win.img.dbg.PushFunction(func() {
						if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
							sk.EEPROM.Poke(uint16(origin+idx), data)
						} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
							vox.SaveKey.EEPROM.Poke(uint16(origin+idx), data)
						}
					})
				},
			)
		}
	}

	if !pagesShown {
		imgui.Text("No pages accessed yet")
	}

	imgui.EndChild()

	win.statusHeight = imguiMeasureHeight(func() {
		imgui.Spacing()
		imgui.Spacing()

		win.showAccessedOnly = win.img.prefs.savekeyAccessPagesOnly.Get().(bool)
		if imgui.Checkbox("Show Accessed Pages Only", &win.showAccessedOnly) {
			win.img.prefs.savekeyAccessPagesOnly.Set(win.showAccessedOnly)
		}
		imgui.Spacing()

		drawDisabled(win.showAccessedOnly, func() {
			if imgui.Button("Jump to Scratchpad") {
				win.scrollScratch = true
			}
		})

		if !win.savekey.EEPROM.IsSaved() {
			imgui.SameLineV(0, 20)
			if imgui.Button("Save to disk") {
				win.img.dbg.PushFunction(func() {
					if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
						sk.EEPROM.Save()
					} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
						vox.SaveKey.EEPROM.Save()
					}
				})
			}

			imgui.SameLineV(0, 5)
			if imgui.Button("Reload") {
				win.img.dbg.PushFunction(func() {
					if sk, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*savekey.SaveKey); ok {
						sk.EEPROM.Restore()
					} else if vox, ok := win.img.dbg.VCS().RIOT.Ports.RightPlayer.(*atarivox.AtariVox); ok {
						vox.SaveKey.EEPROM.Restore()
					}
				})
			}
		}

	})
}
