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

	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/veandco/go-sdl2/sdl"
)

// time periods in milliseconds that each mode sleeps for at the end of each
// service() call. this changes depending primarily on whether we're in debug
// or play mode.
const (
	playSleepPeriod = 0

	// settling on 10ms sleep period for debugger. this strikes a balance
	// between responsiveness and CPU usage.
	//
	// it was set at 50ms for a while but this felt too sluggish. I think when
	// combined with the lazyupdate system, which happens in two half (refresh
	// and update) the delay is visually a lot longer in some situations (for
	// example, when editing the PC value and seeing the change in the disasm
	// window).
	debugSleepPeriod = 10

	// idleSleepPeriod should not be too long because the sleep is not preempted.
	idleSleepPeriod = 500
)

// time periods used to slow down / speed up event handling (in milliseconds).
const (
	// frictionPeriod is applied to mouse events. mouse events can come in
	// thick and fast and we don't want/need to service everyone.
	frictionPeriod = 50

	// the period of inactivity required before the main sleep period drops to
	// the idlsSleepPeriod value.
	wakefullnessPeriod = 3000 // 3 seconds
)

type polling struct {
	img *SdlImgui

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

	// the following are not used in playmode

	// alerted is used to preempt the tickers when we want to communicate between
	// iterations of the service loop. for example, closing sdlimgui windows
	// might feel laggy without it (see commentary in service loop for
	// explanation).
	alerted bool

	// the awake flag is set to true for a short time (defined by
	// wakefullnessPeriod) after the last event was received. this improves
	// responsiveness for certain GUI operations.
	awake     bool
	lastEvent time.Time

	// measure rendering performance
	measureCt             int
	measureTime           time.Time
	measuringPulse        *time.Ticker
	measuredRenderingTime float32
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
		measuringPulse: time.NewTicker(time.Second),
	}

	return pol
}

// alert() forces the next call to wait to resolve immediately.
func (pol *polling) alert() {
	// does nothing in playmode but it's cheaper to just set the flag
	pol.alerted = true
}

// wait for an SDL event or for a timeout. the timeout duration depends on the
// state of the emulation and receent user input.
func (pol *polling) wait() sdl.Event {
	// measure rendering performance
	pol.measureCt++
	select {
	case <-pol.measuringPulse.C:
		t := time.Now()

		pol.measuredRenderingTime = float32(pol.measureCt) / float32(t.Sub(pol.measureTime).Seconds())
		pol.measureTime = t
		pol.measureCt = 0
	default:
	}

	select {
	case f := <-pol.service:
		f()
	case r := <-pol.featureSet:
		pol.img.serviceSetFeature(r)
	case r := <-pol.featureGet:
		pol.img.serviceGetFeature(r)
	default:
	}

	if pol.img.isPlaymode() {
		// wait for new SDL event or until the selected timeout period has elapsed
		return sdl.WaitEventTimeout(playSleepPeriod)
	}

	// decide on timeout period
	var timeout int

	if pol.alerted {
		pol.alerted = false
	} else {
		working := pol.awake ||
			pol.img.lz.Debugger.HasChanged || pol.img.emulation.State() != emulation.Paused ||
			pol.img.wm.dbgScr.crtPreview

		if working {
			timeout = debugSleepPeriod
		} else {
			timeout = idleSleepPeriod
		}
	}

	ev := sdl.WaitEventTimeout(timeout)

	if ev != nil {
		// an event has been received so set awake flag and note time of event
		pol.awake = true
		pol.lastEvent = time.Now()
	} else if pol.awake {
		// keep awake flag set for wakefullnessPeriod milliseconds
		pol.awake = time.Since(pol.lastEvent).Milliseconds() < wakefullnessPeriod
	}

	// slow down mouse events unless input has been "captured". if we don't do
	// this then waggling the mouse over the screen will increase CPU usage
	// significantly. CPU usage will still increase but by a smaller margin.
	if !pol.img.isCaptured() {
		switch ev.(type) {
		case *sdl.MouseMotionEvent:
			time.Sleep(frictionPeriod * time.Millisecond)
		}
	}

	return ev
}
