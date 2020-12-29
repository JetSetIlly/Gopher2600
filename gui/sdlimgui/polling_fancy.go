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

// +build linux darwin

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

	// tickers are the basic construct in the polling type. they prevent the
	// service loop from iterating too quickly.
	//
	// which ticker is used depends on the whether we're in playmode and the
	// state of the debugger.
	play *time.Ticker
	dbg  *time.Ticker
	idle *time.Ticker

	// events are captured parallel to the main service loop and used to
	// preempt the selected ticker.
	event chan sdl.Event

	// wake is used to preempt the tickers when we want to communicate between
	// iterations of the service loop. for example, closing sdlimgui windows
	// might feel laggy without it (see commentary in service loop for
	// explanation).
	//
	// because sending and receiving to this channel is likely to happen in the
	// same goroutine then care needs to be taken not to cause a deadlock. use
	// alert() function to prevent deadlock.
	wake chan bool
}

func newPolling(img *SdlImgui) *polling {
	pol := &polling{
		img:   img,
		event: make(chan sdl.Event, 1),
		wake:  make(chan bool, 1),
	}

	// initialise tickers
	pol.dbg = time.NewTicker(time.Millisecond * debugSleepPeriod)
	pol.play = time.NewTicker(time.Millisecond * playSleepPeriod)
	pol.idle = time.NewTicker(time.Millisecond * idleSleepPeriod)

	// loop and wait for SDL events. it is this construct that does not work on
	// some operating systems (eg. Windows) - I think because we're calling a
	// sdl function in a non-main goroutine
	go func() {
		for {
			pol.event <- sdl.WaitEvent()
		}
	}()

	return pol
}

// alert() forces the next call to wait to resolve immediately.
func (pol *polling) alert() {
	// pushing event inside a select/default block to prevent channel deadlock
	// - send and retrieve to the channel happens in the same goroutine
	select {
	case pol.wake <- true:
	default:
	}
}

func (pol *polling) wait() sdl.Event {
	var ev sdl.Event

	if pol.img.isPlaymode() {
		select {
		case <-pol.play.C: // timeout
		case <-pol.wake:
		case ev = <-pol.event:
		case r := <-pol.img.featureSet:
			pol.img.serviceSetFeature(r)
		case r := <-pol.img.featureGet:
			pol.img.serviceGetFeature(r)
		}

		return ev
	}

	// we know we're in the debugger but we must still decide which timeout ticker to use.
	var pulse <-chan time.Time

	// the positive branch selects the more frequent ticker (ie. the one
	// that leads to more CPU usage).
	//
	// we trigger this when the debugger thinks something has changed: when the
	// emulation is running or when a CRT effect is active.
	//
	// the CRT conditions are required because one of the CRT effects is an
	// animated effect (the noise generator), which requires frequent updates.
	if pol.img.lz.Debugger.HasChanged || pol.img.state == gui.StateRunning || pol.img.wm.dbgScr.crt || pol.img.wm.crtPrefs.open {
		pulse = pol.dbg.C
	} else {
		// if nothing much is happening then we use the slowest pulse ticker
		pulse = pol.idle.C
	}

	select {
	case <-pulse:
	case <-pol.wake:
	case ev = <-pol.event:
		// slow down mouse events unless input has been "captured".
		//
		// if we allow every mouse motion event to preempt the pulse ticker
		// then we can increase CPU usage simply by waggling the mouse.
		//
		// if mouse has been captured then we treat mouse events the same as in
		// playmode.
		if !pol.img.isCaptured() {
			switch ev.(type) {
			case *sdl.MouseMotionEvent:
				<-pol.dbg.C
			}
		}
	case r := <-pol.img.featureSet:
		pol.img.serviceSetFeature(r)
	case r := <-pol.img.featureGet:
		pol.img.serviceGetFeature(r)
	}

	return ev
}
