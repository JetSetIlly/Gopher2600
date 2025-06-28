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
	"strings"

	"github.com/jetsetilly/imgui-go/v5"
)

var tooltipColor = imgui.Vec4{X: 0.08, Y: 0.08, Z: 0.08, W: 1.0}
var tooltipBorder = imgui.Vec4{X: 0.03, Y: 0.03, Z: 0.03, W: 1.0}
var tooltipAlpha = 0.83

// shows tooltip on hover of the previous imgui digest/group. useful for simple
// tooltips.
func (img *SdlImgui) imguiTooltipSimple(tooltip string) {
	show := img.prefs.showTooltips.Get().(bool) || imgui.CurrentIO().KeyCtrlPressed()
	img.tooltipIndicator = imguiTooltipSimple(tooltip, show) || img.tooltipIndicator
}

// shows simple tooltip but without global preferences test
//
// returns true if the tooltip would have been displayed except for the show
// flag. ie. if show is false the function only tests whether the tooltip would
// have shown
func imguiTooltipSimple(tooltip string, show bool) bool {
	var displayed bool

	// we always want to draw tooltips with the tooltip alpha
	imgui.PushStyleVarFloat(imgui.StyleVarAlpha, float32(tooltipAlpha))
	defer imgui.PopStyleVar()

	// split string on newline and display with separate calls to imgui.Text()
	tooltip = strings.TrimSpace(tooltip)
	if tooltip != "" && imgui.IsItemHovered() {
		displayed = true

		if show {
			s := strings.Split(tooltip, "\n")
			imgui.PushStyleColor(imgui.StyleColorPopupBg, tooltipColor)
			imgui.PushStyleColor(imgui.StyleColorBorder, tooltipBorder)
			imgui.BeginTooltip()
			for _, t := range s {
				imgui.Text(t)
			}
			imgui.EndTooltip()
			imgui.PopStyleColorV(2)
		}
	}

	return displayed
}

// shows tooltip that require more complex formatting than a single string.
//
// the hoverTest argument says that the tooltip should be displayed only
// when the last imgui widget/group is being hovered over with the mouse. if
// this argument is false then it implies that the decision to show the tooltip
// has already been made by the calling function.
func (img *SdlImgui) imguiTooltip(f func(), hoverTest bool) {
	show := img.prefs.showTooltips.Get().(bool) || imgui.CurrentIO().KeyCtrlPressed()
	img.tooltipIndicator = imguiTooltip(f, hoverTest, show) || img.tooltipIndicator
}

// shows tooltip but without global preferences test
//
// returns true if the tooltip would have been displayed except for the show
// flag. ie. if show is false the function only tests whether the tooltip would
// have shown
func imguiTooltip(f func(), hoverTest bool, show bool) bool {
	var displayed bool

	// we always want to draw tooltips with the tooltip alpha
	imgui.PushStyleVarFloat(imgui.StyleVarAlpha, float32(tooltipAlpha))
	defer imgui.PopStyleVar()

	if !hoverTest || imgui.IsItemHovered() {
		displayed = true

		if show {
			imgui.PushStyleColor(imgui.StyleColorPopupBg, tooltipColor)
			imgui.PushStyleColor(imgui.StyleColorBorder, tooltipBorder)
			imgui.BeginTooltip()
			f()
			imgui.EndTooltip()
			imgui.PopStyleColorV(2)
		}
	}

	return displayed
}
