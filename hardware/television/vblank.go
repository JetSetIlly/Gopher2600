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

package television

import "github.com/jetsetilly/gopher2600/hardware/television/signal"

type vblankPhase int

const (
	phaseTop vblankPhase = iota
	phaseMiddle
	phaseBottom
)

// vblankBounds is similar to the Resizer except that it only deals with the
// VBLANK signal and doesn't care about the physical size of the screen
type vblankBounds struct {
	phase  vblankPhase
	top    int
	bottom int
	vblank bool
}

func (b *vblankBounds) reset() {
	b.phase = phaseTop
	b.top = -1
	b.bottom = -1
	b.vblank = false
}

func (b *vblankBounds) examine(sig signal.SignalAttributes, scanline int) {
	vblank := sig&signal.VBlank == signal.VBlank

	switch b.phase {
	case phaseTop:
		if b.vblank && !vblank {
			b.top = scanline
			b.phase = phaseMiddle
		}
	case phaseMiddle:
		if !b.vblank && vblank {
			b.bottom = scanline
			b.phase = phaseBottom
		}
	case phaseBottom:
		if b.vblank && !vblank {
			b.phase = phaseMiddle
		}
	}

	b.vblank = vblank
}

func (b *vblankBounds) commit(state *State) bool {
	var changed bool

	if state.frameInfo.Stable && state.vsync.isSynced() {
		changed = state.frameInfo.VBLANKtop != b.top || state.frameInfo.VBLANKbottom != b.bottom
		state.frameInfo.VBLANKunstable = state.frameInfo.VBLANKunstable || changed
	}

	state.frameInfo.VBLANKtop = b.top
	state.frameInfo.VBLANKbottom = b.bottom
	state.frameInfo.VBLANKatari = b.top == state.frameInfo.Spec.AtariSafeVisibleTop &&
		b.bottom == state.frameInfo.Spec.AtariSafeVisibleBottom

	b.reset()

	return changed
}
