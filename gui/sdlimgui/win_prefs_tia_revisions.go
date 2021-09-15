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
)

func (win *winPrefs) drawTIARev() {
	imgui.Spacing()

	if imgui.BeginTableV("tiaRevisions", 2, imgui.TableFlagsBordersInnerV, imgui.Vec2{}, 1.0) {
		imgui.TableNextRow()
		imgui.TableNextColumn()

		win.drawLateGRPx()
		win.drawRESPxUnderHMOVE()

		imgui.TableNextColumn()

		win.drawLatePlayfield()
		win.drawLostMOTCK()
		win.drawLateRESPx()

		imgui.EndTable()
	}
}

func (win *winPrefs) drawTIARevTooltip(bug revision.Bug) {
	imguiTooltipSimple(fmt.Sprintf("%s\nNotable ROM: %s", bug.Description(), bug.NotableROM()))
}

func (win *winPrefs) drawLateGRPx() {
	imgui.Spacing()
	imgui.Text("Late VDEL gfx")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP0.Get().(bool)
	if imgui.Checkbox("GRP0", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP0.Set(a)
	}
	win.drawTIARevTooltip(revision.LateVDELGRP0)

	b := win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP1.Get().(bool)
	if imgui.Checkbox("GRP1", &b) {
		win.img.vcs.TIA.Rev.Prefs.DskLateVDELGRP1.Set(b)
	}
	win.drawTIARevTooltip(revision.LateVDELGRP1)
}

func (win *winPrefs) drawRESPxUnderHMOVE() {
	imgui.Spacing()
	imgui.Text("RESPx under HMOVE")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLateRESPx.Get().(bool)
	if imgui.Checkbox("Late RESPx", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLateRESPx.Set(a)
	}
	win.drawTIARevTooltip(revision.LateRESPx)

	b := win.img.vcs.TIA.Rev.Prefs.DskEarlyScancounter.Get().(bool)
	if imgui.Checkbox("Early Scancounter", &b) {
		win.img.vcs.TIA.Rev.Prefs.DskEarlyScancounter.Set(b)
	}
	win.drawTIARevTooltip(revision.EarlyScancounter)
}

func (win *winPrefs) drawLatePlayfield() {
	imgui.Spacing()
	imgui.Text("Late Playfield")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLatePFx.Get().(bool)
	if imgui.Checkbox("PFx", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLatePFx.Set(a)
	}
	win.drawTIARevTooltip(revision.LatePFx)

	b := win.img.vcs.TIA.Rev.Prefs.DskLateCOLUPF.Get().(bool)
	if imgui.Checkbox("COLUPF", &b) {
		win.img.vcs.TIA.Rev.Prefs.DskLateCOLUPF.Set(b)
	}
	win.drawTIARevTooltip(revision.LateCOLUPF)
}

func (win *winPrefs) drawLostMOTCK() {
	imgui.Spacing()
	imgui.Text("Lost MOTCK")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskLostMOTCK.Get().(bool)
	if imgui.Checkbox("Players/Missiles/Ball", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskLostMOTCK.Set(a)
	}
	win.drawTIARevTooltip(revision.LostMOTCK)
}

func (win *winPrefs) drawLateRESPx() {
	imgui.Spacing()
	imgui.Text("RESPx")
	imgui.Spacing()
	a := win.img.vcs.TIA.Rev.Prefs.DskRESPxHBLANK.Get().(bool)
	if imgui.Checkbox("HBLANK threshold", &a) {
		win.img.vcs.TIA.Rev.Prefs.DskRESPxHBLANK.Set(a)
	}
	win.drawTIARevTooltip(revision.RESPxHBLANK)
}
