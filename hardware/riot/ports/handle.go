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

package ports

import (
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
)

func (p *Ports) HandleInputEvents() error {
	err := p.handlePushedEvents()
	if err != nil {
		return err
	}

	err = p.handleDrivenEvents()
	if err != nil {
		return err
	}

	err = p.handlePlaybackEvents()
	if err != nil {
		return err
	}

	return nil
}

func (p *Ports) handlePushedEvents() error {
	done := false
	for !done {
		select {
		case inp := <-p.pushed:
			_, err := p.HandleInputEvent(inp)
			if err != nil {
				return err
			}
		default:
			done = true
		}
	}
	return nil
}

// handleDrivenEvents checks for input driven from a another emulation.
func (p *Ports) handleDrivenEvents() error {
	if p.checkForDriven {
		inp := p.drivenInputData
		done := false
		for !done {
			c := p.tv.GetCoords()
			if coords.Equal(c, inp.Time) {
				_, err := p.HandleInputEvent(inp.InputEvent)
				if err != nil {
					return err
				}
			} else if coords.GreaterThan(c, inp.Time) {
				return curated.Errorf("ports: driven input seen too late. emulations not synced correctly.")
			} else {
				return nil
			}

			select {
			case p.drivenInputData = <-p.fromDriver:
				if p.checkForDriven {
					curated.Errorf("ports: driven input received before previous input was processed")
				}
			default:
				done = true
				p.checkForDriven = false
			}

			inp = p.drivenInputData
		}
	}

	if p.fromDriver != nil {
		select {
		case p.drivenInputData = <-p.fromDriver:
			if p.checkForDriven {
				curated.Errorf("ports: driven input received before previous input was processed")
			}
			p.checkForDriven = true
		default:
		}
	}

	return nil
}

// handlePlaybackEvents requests playback events from all attached and eligible peripherals.
func (p *Ports) handlePlaybackEvents() error {
	if p.playback == nil {
		return nil
	}

	// loop with GetPlayback() until we encounter a NoPortID or NoEvent
	// condition. there might be more than one entry for a particular
	// frame/scanline/horizpas state so we need to make sure we've processed
	// them all.
	//
	// this happens in particular with recordings that were made of  ROMs with
	// panel setup configurations (see setup package) - where the switches are
	// set when the TV state is at fr=0 sl=0 cl=0
	morePlayback := true
	for morePlayback {
		inp, err := p.playback.GetPlayback()
		if err != nil {
			return err
		}

		morePlayback = inp.Port != plugging.PortUnplugged && inp.Ev != NoEvent
		if morePlayback {
			_, err := p.HandleInputEvent(inp.InputEvent)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// HandleInputEvent should only be used from the same gorouine as the
// emulation. Events should be queued with QueueEvent() otherwise.
//
// Consider using HandleInputEvent() function in the VCS type rather than this
// function directly.
func (p *Ports) HandleInputEvent(inp InputEvent) (bool, error) {
	var handled bool
	var err error

	switch inp.Port {
	case plugging.PortPanel:
		handled, err = p.Panel.HandleEvent(inp.Ev, inp.D)
	case plugging.PortLeftPlayer:
		handled, err = p.LeftPlayer.HandleEvent(inp.Ev, inp.D)
	case plugging.PortRightPlayer:
		handled, err = p.RightPlayer.HandleEvent(inp.Ev, inp.D)
	}

	// forward to passenger if one is defined
	if handled && p.toPassenger != nil {
		select {
		case p.toPassenger <- TimedInputEvent{Time: p.tv.GetCoords(), InputEvent: inp}:
		default:
			return handled, curated.Errorf("ports: passenger event queue is full: input dropped")
		}
	}

	// if error was because of an unhandled event then return without error
	if err != nil {
		return handled, curated.Errorf("ports: %v", err)
	}

	// record event with the EventRecorder
	for _, r := range p.recorder {
		return handled, r.RecordEvent(TimedInputEvent{Time: p.tv.GetCoords(), InputEvent: inp})
	}

	return handled, nil
}

// QueueEvent pushes an InputEvent onto the queue. Will drop the event and
// return an error if queue is full.
func (p *Ports) QueueEvent(inp InputEvent) error {
	select {
	case p.pushed <- inp:
	default:
		return curated.Errorf("ports: pushed event queue is full: input dropped")
	}
	return nil
}
