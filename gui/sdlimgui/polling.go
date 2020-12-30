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
	"time"

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/veandco/go-sdl2/sdl"
)

// time periods in milliseconds that each mode sleeps for at the end of each
// service() call. this changes depending on whether we're in debug or play
// mode.
const (
	debugSleepPeriod = 50
	playSleepPeriod  = 10
	idleSleepPeriod  = 500
)

type polling struct {
	img *SdlImgui

	dbgTicker *time.Ticker

	// wake is used to preempt the tickers when we want to communicate between
	// iterations of the service loop. for example, closing sdlimgui windows
	// might feel laggy without it (see commentary in service loop for
	// explanation).
	wake bool

	// functions that need to be performed in the main thread are queued for
	// serving by the service() function
	service    chan func()
	serviceErr chan error

	// ReqFeature() and GetFeature() hands off requests to the featureReq
	// channel for servicing. think of these as pecial instances of the
	// service chan
	featureSet     chan featureRequest
	featureSetErr  chan error
	featureGet     chan featureRequest
	featureGetData chan gui.FeatureReqData
	featureGetErr  chan error
}

func newPolling(img *SdlImgui) *polling {
	pol := &polling{
		img:            img,
		service:        make(chan func(), 1),
		serviceErr:     make(chan error, 1),
		featureSet:     make(chan featureRequest, 1),
		featureSetErr:  make(chan error, 1),
		featureGet:     make(chan featureRequest, 1),
		featureGetData: make(chan gui.FeatureReqData, 1),
		featureGetErr:  make(chan error, 1),
	}

	pol.dbgTicker = time.NewTicker(time.Millisecond * debugSleepPeriod)

	return pol
}

// alert() forces the next call to wait to resolve immediately.
func (pol *polling) alert() {
	pol.wake = true
}

func (pol *polling) wait() sdl.Event {
	select {
	case f := <-pol.service:
		f()
	case r := <-pol.featureSet:
		pol.img.serviceSetFeature(r)
	case r := <-pol.featureGet:
		pol.img.serviceGetFeature(r)
	default:
	}

	var timeout int

	if pol.wake {
		pol.wake = false
	} else {
		if pol.img.isPlaymode() {
			timeout = playSleepPeriod
		} else {
			// the positive branch selects the more frequent ticker (ie. the one
			// that leads to more CPU usage).
			//
			// we trigger this when the debugger thinks something has changed: when the
			// emulation is running or when a CRT effect is active.
			//
			// the CRT conditions are required because one of the CRT effects is an
			// animated effect (the noise generator), which requires frequent updates.
			if pol.img.lz.Debugger.HasChanged || pol.img.state == gui.StateRunning || pol.img.wm.dbgScr.crt || pol.img.wm.crtPrefs.open || pol.img.state == gui.StateInitialising {
				timeout = debugSleepPeriod
			} else {
				timeout = idleSleepPeriod
			}
		}
	}

	// wait for new SDL event or until the selected timeout period has elapsed
	ev := sdl.WaitEventTimeout(timeout)

	// slow down mouse events unless input has been "captured". if we don't do
	// this then waggling the mouse over the screen will increase CPU usage
	// significantly. CPU usage will still increase but by a smaller margin.
	if !pol.img.isCaptured() {
		switch ev.(type) {
		case *sdl.MouseMotionEvent:
			<-pol.dbgTicker.C
		}
	}

	return ev
}
