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

package shaders

import _ "embed"

//go:embed "straight.vert"
var StraightVertexShader []byte

//go:embed "yflip.vert"
var YFlipVertexShader []byte

//go:embed "gui.frag"
var GUIShader []byte

//go:embed "color.frag"
var ColorShader []byte

//go:embed "dbgscr.frag"
var DbgScrShader []byte

//go:embed "dbgscr_overlay.frag"
var DbgScrOverlayShader []byte

//go:embed "dbgscr_helpers.frag"
var DbgScrHelpersShader []byte

//go:embed "sharpen.frag"
var SharpenShader []byte

//go:embed "crt_effects.frag"
var CRTEffectsFragShader []byte

//go:embed "crt_blur.frag"
var CRTBlurFragShader []byte

//go:embed "crt_ghosting.frag"
var CRTGhostingFragShader []byte

//go:embed "crt_phosphor.frag"
var CRTPhosphorFragShader []byte

//go:embed "crt_blackcorrection.frag"
var CRTBlackCorrection []byte

//go:embed "crt_screenroll.frag"
var CRTScreenroll []byte
