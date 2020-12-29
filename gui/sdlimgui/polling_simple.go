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

// +build !linux,!darwin

package sdlimgui

import (
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// time periods in milliseconds that each mode sleeps for at the end of each
// service() call. this changes depending on whether we're in debug or play
// mode.
const (
	debugSleepPeriod = 50
	playSleepPeriod  = 10
)

type polling struct {
	img *SdlImgui

	// tickers are the basic construct in the polling type. they prevent the
	// service loop from iterating too quickly.
	//
	// which ticker is used depends on the whether we're in playmode and the
	// state of the debugger.
	play *time.Ticker
	dbg  *time.Ticker

	// wake is used to preempt the tickers when we want to communicate between
	// iterations of the service loop. for example, closing sdlimgui windows
	// might feel laggy without it (see commentary in service loop for
	// explanation).
	wake bool
}

func newPolling(img *SdlImgui) *polling {
	pol := &polling{
		img: img,
	}

	// initialise tickers
	pol.dbg = time.NewTicker(time.Millisecond * debugSleepPeriod)
	pol.play = time.NewTicker(time.Millisecond * playSleepPeriod)

	return pol
}

// alert() forces the next call to wait to resolve immediately.
func (pol *polling) alert() {
	pol.wake = true
}

func (pol *polling) wait() sdl.Event {
	if pol.wake {
		pol.wake = false
	} else {
		var pulse <-chan time.Time

		if pol.img.isPlaymode() {
			pulse = pol.play.C
		} else {
			pulse = pol.dbg.C
		}

		select {
		case <-pulse: // timeout
		case r := <-pol.img.featureSet:
			pol.img.serviceSetFeature(r)
		case r := <-pol.img.featureGet:
			pol.img.serviceGetFeature(r)
		}
	}

	return sdl.PollEvent()
}
