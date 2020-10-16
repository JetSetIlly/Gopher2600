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

package video

import (
	"fmt"
	"strings"
)

// enclockifier is the mechanism controlling how many pixels to output for both
// ball and missile sprites. it is the equivalent to the scanCounter mechanism
// used by the player sprite.
//
// the peculiar name is taken from TIA_HW_Notes:
//
// "Notes on the Ball/Missile width enclockifier
//
// Just to reiterate, ball width is given by combining clock signals
// of different widths based on the state of the two size bits (the
// gates form an AND -> OR -> AND -> OR -> out arrangement, with a
// hanger-on AND gate).".
type enclockifier struct {
	Active     bool
	SecondHalf bool
	Ticks      int
	Paused     bool

	// which copy of the sprite is being drawn (ball sprite only ever has one
	// copy). value of zero means the primary copy is being drawn (if enable is
	// true)
	Cpy int

	// size of ball/missile
	size *uint8
}

func (en *enclockifier) String() string {
	s := strings.Builder{}
	if en.Active {
		s.WriteString(fmt.Sprintf("%d", en.Ticks))
		if en.SecondHalf {
			s.WriteString("/2nd")
		}
		s.WriteString(")")

		if en.Cpy > 0 {
			s.WriteString(fmt.Sprintf("+%d", en.Cpy))
		}
	}
	return s.String()
}

// the ball sprite drops enclockifier events during position resets.
func (en *enclockifier) drop() {
	en.Active = false
}

// the ball sprite forces conclusion (or continuation in the case of 8x width)
// of enclockifier events during position resets.
func (en *enclockifier) force() {
	en.Paused = false
	if *en.size == 0x03 && en.Active && !en.SecondHalf {
		en.SecondHalf = true
		en.Ticks = 0
	} else {
		en.Active = false
	}
}

func (en *enclockifier) aboutToEnd() bool {
	if !en.Active {
		return false
	}
	switch *en.size {
	case 0x00:
		return en.Ticks == 0
	case 0x01:
		return en.Ticks == 1
	case 0x02:
		return en.Ticks == 3
	case 0x03:
		return en.Ticks == 3
	}
	return false
}

func (en *enclockifier) start() {
	en.Active = true
	en.Paused = false
	en.SecondHalf = false
	en.Ticks = 0
}

func (en *enclockifier) tick() {
	if !en.Active || en.Paused {
		return
	}

	en.Ticks++
	switch *en.size {
	case 0x00:
		if en.Ticks >= 1 || en.SecondHalf {
			en.Active = false
		}
	case 0x01:
		if en.Ticks >= 2 || en.SecondHalf {
			en.Active = false
		}
	case 0x02:
		if en.Ticks >= 4 || en.SecondHalf {
			en.Active = false
		}
	case 0x03:
		if en.Ticks >= 4 {
			if en.SecondHalf {
				en.Active = false
			} else {
				en.SecondHalf = true
				en.Ticks = 0
			}
		}
	}
}
