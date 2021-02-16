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

	"github.com/inkyblackness/imgui-go/v4"
	"github.com/jetsetilly/gopher2600/hardware/tia/revision"
	"github.com/jetsetilly/gopher2600/logger"
)

const winTIARevisionsID = "TIA Revisions"

type winTIARevisions struct {
	img  *SdlImgui
	open bool
}

func newWinRevisions(img *SdlImgui) (window, error) {
	win := &winTIARevisions{
		img: img,
	}

	return win, nil
}

func (win *winTIARevisions) init() {
}

func (win *winTIARevisions) id() string {
	return winTIARevisionsID
}

func (win *winTIARevisions) isOpen() bool {
	return win.open
}

func (win *winTIARevisions) setOpen(open bool) {
	win.open = open
}

func (win *winTIARevisions) draw() {
	if !win.open {
		return
	}

	if win.img.isPlaymode() {
		imgui.SetNextWindowPosV(imgui.Vec2{25, 25}, imgui.ConditionAppearing, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsNoSavedSettings|imgui.WindowFlagsAlwaysAutoResize)
	} else {
		imgui.SetNextWindowPosV(imgui.Vec2{25, 25}, imgui.ConditionFirstUseEver, imgui.Vec2{0, 0})
		imgui.BeginV(win.id(), &win.open, imgui.WindowFlagsAlwaysAutoResize)
	}

	win.drawLateGRPx()
	imguiSeparator()
	win.drawLateRipple()
	imguiSeparator()
	win.drawLatePlayfield()
	imguiSeparator()
	win.drawLostMOTCK()
	imguiSeparator()
	win.drawLateRESPx()

	imguiSeparator()
	win.drawDiskButtons()

	imgui.End()
}

func (win *winTIARevisions) drawTooltip(bug revision.Bug) {
	imgui.BeginTooltip()
	defer imgui.EndTooltip()
	imgui.Text(bug.Description())
	imgui.Text("Notable ROM:")
	imgui.SameLine()
	imgui.Text(bug.NotableROM())
}

func (win *winTIARevisions) drawLateGRPx() {
	imgui.Text("Late VDEL gfx")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP0.Get().(bool)
	if imgui.Checkbox("GRP0", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP0.Set(a)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LateVDELGRP0)
	}

	b := win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP1.Get().(bool)
	if imgui.Checkbox("GRP1", &b) {
		win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP1.Set(b)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LateVDELGRP1)
	}
}

func (win *winTIARevisions) drawLateRipple() {
	imgui.Text("HMOVE (ripple)")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLateRippleStart.Get().(bool)
	if imgui.Checkbox("Late Start", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLateRippleStart.Set(a)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LateRippleStart)
	}

	b := win.img.vcs.TIA.Rev.Prefs.DskLateRippleEnd.Get().(bool)
	if imgui.Checkbox("Late End", &b) {
		win.img.vcs.TIA.Rev.Prefs.DskLateRippleEnd.Set(b)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LateRippleEnd)
	}
}

func (win *winTIARevisions) drawLatePlayfield() {
	imgui.Text("Late Playfield")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLatePFx.Get().(bool)
	if imgui.Checkbox("PFx", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLatePFx.Set(a)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LatePFx)
	}

	b := win.img.vcs.TIA.Rev.Prefs.DskLateCOLUPF.Get().(bool)
	if imgui.Checkbox("COLUPF", &b) {
		win.img.vcs.TIA.Rev.Prefs.DskLateCOLUPF.Set(b)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LateCOLUPF)
	}
}

func (win *winTIARevisions) drawLostMOTCK() {
	imgui.Text("Lost MOTCK")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLostMOTCK.Get().(bool)
	if imgui.Checkbox("Players/Missiles/Ball", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLostMOTCK.Set(a)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.LostMOTCK)
	}
}

func (win *winTIARevisions) drawLateRESPx() {
	imgui.Text("RESPx")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskRESPxHBLANK.Get().(bool)
	if imgui.Checkbox("HBLANK threshold", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskRESPxHBLANK.Set(a)
	}
	if imgui.IsItemHovered() {
		win.drawTooltip(revision.RESPxHBLANK)
	}
}

func (win *winTIARevisions) drawDiskButtons() {
	if imgui.Button("Save") {
		err := win.img.vcs.TIA.Rev.Prefs.Save()
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not save tia revision settings: %v", err))
		}
	}

	imgui.SameLine()
	if imgui.Button("Restore") {
		err := win.img.vcs.TIA.Rev.Prefs.Load()
		if err != nil {
			logger.Log("sdlimgui", fmt.Sprintf("could not restore tia revision settings: %v", err))
		}
	}
}
