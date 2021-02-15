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

// Package hmove represents the TIA HMOVE process.
package hmove

import (
	"fmt"
	"strings"

	"github.com/jetsetilly/gopher2600/hardware/tia/delay"
)

type Hmove struct {
	// the delay between HMOVE being triggered and the latch flag being set
	// to true
	FutureLatch delay.Event

	// Latch indicates whether HMOVE has been triggered this scanline. it is
	// reset when a new scanline begins
	Latch bool

	// the delay between HMOVE being triggered and the ripple count starting
	Future delay.Event

	// Ripple counts backwards from 15 to -1 (represented by 255). note that
	// unlike how it is described in TIA_HW_Notes.txt, we always send the extra
	// tick to the sprites on Phi1.  however, we also send the HmoveCt value,
	// whether or not the extra should be honoured is up to the sprite.
	// (TIA_HW_Notes.txt says that HmoveCt is checked *before* sending the
	// extra tick)
	Ripple uint8

	// Has the Ripple counter just expired this cycle
	RippleJustEnded bool

	// Clk is true when the TIA PhaseClock.Phi2() is true
	Clk bool
}

// ResetRipple begins the ripple count.
func (hm *Hmove) ResetRipple() {
	hm.Ripple = 15
}

// Tick every video cycle when Clk is true. ie. when the TIA phaseclock is in
// rising Phi2.
func (hm *Hmove) Tick() {
	hm.RippleJustEnded = false
	if hm.Ripple != 0xff {
		hm.Ripple--
		hm.RippleJustEnded = hm.Ripple == 0xff
	}
}

// Reset Hmove values.
func (hm *Hmove) Reset() {
	hm.Latch = false
	hm.Ripple = 0xff
	hm.Clk = false
	hm.FutureLatch.Drop()
	hm.Future.Drop()
}

func (hm *Hmove) String() string {
	s := strings.Builder{}

	if hm.Clk {
		s.WriteString("[Clk]")
	}

	if hm.Future.IsActive() {
		s.WriteString(fmt.Sprintf(" HMOVE latching %d", hm.Future.Remaining()))
	} else if hm.Latch {
		s.WriteString(" HMOVE latched")
	} else {
		s.WriteString(" HMOVE not latched")
	}

	if hm.Ripple <= 15 {
		s.WriteString(fmt.Sprintf(" (ripple count %d)", hm.Ripple))
	} else if hm.RippleJustEnded {
		s.WriteString(" (ripple just ended)")
	}

	return strings.TrimSpace(s.String())
}
