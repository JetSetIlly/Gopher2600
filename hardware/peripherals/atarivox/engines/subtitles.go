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

package engines

import (
	"io"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox/msa"
)

type Subtitles struct {
	w     io.Writer
	reset chan bool
	quit  chan bool
}

// NewFestival creats a new subtitles instance
func NewSubtitles(w io.Writer) *Subtitles {
	sub := &Subtitles{
		w:     w,
		reset: make(chan bool, 1),
		quit:  make(chan bool, 1),
	}

	// amount of time to wait before outputting a newline
	const delay = 1 * time.Second

	go func() {
		timer := time.NewTimer(0)
		timer.Stop()
		for {
			select {
			case <-sub.reset:
				timer.Reset(delay)
			case <-sub.quit:
				timer.Stop()
				return
			case <-timer.C:
				w.Write([]byte("\n"))
			}
		}
	}()

	return sub
}

// Quit implements the AtariVoxEngine interface.
func (sub *Subtitles) Quit() {
	select {
	case sub.quit <- true:
	default:
	}
}

// SpeakJet implements the AtariVoxEngine interface.
func (sub *Subtitles) SpeakJet(command uint8, _ uint8) {
	if a, ok := msa.Commands[command].(msa.Allophone); ok {
		p := msa.Phonetics[a.Phoneme]
		sub.w.Write([]byte(p.Phonetic))
		select {
		case sub.reset <- true:
		default:
		}
	}
}

// Flush implements the AtariVoxEngine interface.
func (sub *Subtitles) Flush() {
}
