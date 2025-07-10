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
	"time"

	"github.com/jetsetilly/gopher2600/hardware/peripherals/atarivox/msa"
	"github.com/jetsetilly/gopher2600/notifications"
)

type subtitle struct {
	phoneme string
	delay   time.Duration
}

type Subtitles struct {
	notify notifications.Notify
	push   chan subtitle
	quit   chan bool
}

const (
	// Enough time has elapsed to consider the subtitle to be "stale". Visualisers should be cleared
	StaleSubtitle = "\n"

	// AtariVox was idle long enough to consider the preceding text to be a single "sentence"
	SubtitleSentence = "\r"

	// the phoneme that indicates a possible space in the subtitles
	spacePhoneme = " "
)

// NewFestival creats a new subtitles instance
func NewSubtitles(notify notifications.Notify) *Subtitles {
	sub := &Subtitles{
		notify: notify,
		push:   make(chan subtitle, 100), // generous queue required because of artificial delay
		quit:   make(chan bool, 1),
	}

	const (
		newlineDelay = 3 * time.Second
		nextDelay    = 250 * time.Millisecond
	)

	go func() {
		newlineTimer := time.NewTimer(0)
		newlineTimer.Stop()
		nextTimer := time.NewTimer(0)
		nextTimer.Stop()

		var lastPhonemeWasSpace bool

		for {
			select {
			case s := <-sub.push:
				if s.phoneme == spacePhoneme {
					if !lastPhonemeWasSpace {
						sub.notify.PushNotify(notifications.NotifyAtariVoxSubtitle, spacePhoneme)
					}
					lastPhonemeWasSpace = true
				} else {
					sub.notify.PushNotify(notifications.NotifyAtariVoxSubtitle, s.phoneme)
					lastPhonemeWasSpace = false
				}
				time.Sleep(s.delay)
				newlineTimer.Reset(newlineDelay)
				nextTimer.Reset(nextDelay)
			case <-sub.quit:
				newlineTimer.Stop()
				nextTimer.Stop()
				return
			case <-newlineTimer.C:
				sub.notify.PushNotify(notifications.NotifyAtariVoxSubtitle, StaleSubtitle)
			case <-nextTimer.C:
				sub.notify.PushNotify(notifications.NotifyAtariVoxSubtitle, SubtitleSentence)
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
func (sub *Subtitles) SpeakJet(command uint8, data uint8) {
	switch cmd := msa.Commands[command].(type) {
	case msa.Allophone:
		p := msa.Phonetics[cmd.Phoneme]
		select {
		case sub.push <- subtitle{
			phoneme: p.Phonetic,
			delay:   time.Duration(cmd.MSec) * time.Millisecond,
		}:
		default:
		}
	case msa.ControlCode:
		switch cmd.Code {
		case 0, 1, 2, 3, 4, 5, 6:
			// pauses
			select {
			case sub.push <- subtitle{
				phoneme: spacePhoneme,
				delay:   time.Duration(msa.PauseLengths[cmd.Code]) * time.Millisecond,
			}:
			default:
			}
		case 30:
			// delay
			select {
			case sub.push <- subtitle{
				phoneme: spacePhoneme,
				delay:   time.Duration(data) * time.Millisecond,
			}:
			default:
			}
		}
	}

}

// Flush implements the AtariVoxEngine interface.
func (sub *Subtitles) Flush() {
}
