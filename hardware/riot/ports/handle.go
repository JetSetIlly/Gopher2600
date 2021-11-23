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

// HandleDrivenEvents checks for input driven from a another emulation.
func (p *Ports) HandleDrivenEvents() error {
	if p.checkForDriven {
		f := p.drivenInputData
		done := false
		for !done {
			c := p.tv.GetCoords()
			if coords.Equal(c, f.time) {
				_, err := p.HandleEvent(f.id, f.ev, f.d)
				if err != nil {
					return err
				}
			} else if coords.GreaterThan(c, f.time) {
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

			f = p.drivenInputData
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

// HandlePlaybackEvents requests playback events from all attached and eligible peripherals.
func (p *Ports) HandlePlaybackEvents() error {
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
		id, ev, v, err := p.playback.GetPlayback()
		if err != nil {
			return err
		}

		morePlayback = id != plugging.PortUnplugged && ev != NoEvent
		if morePlayback {
			_, err := p.HandleEvent(id, ev, v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// HandleEvent implements userinput.HandleInput interface.
func (p *Ports) HandleEvent(id plugging.PortID, ev Event, d EventData) (bool, error) {
	var handled bool
	var err error

	switch id {
	case plugging.PortPanel:
		handled, err = p.Panel.HandleEvent(ev, d)
	case plugging.PortLeftPlayer:
		handled, err = p.LeftPlayer.HandleEvent(ev, d)
	case plugging.PortRightPlayer:
		handled, err = p.RightPlayer.HandleEvent(ev, d)
	}

	if handled && p.toPassenger != nil {
		select {
		case p.toPassenger <- DrivenEvent{time: p.tv.GetCoords(), id: id, ev: ev, d: d}:
		default:
			return handled, curated.Errorf("ports: %v", err)
		}
	}

	// if error was because of an unhandled event then return without error
	if err != nil {
		return handled, curated.Errorf("ports: %v", err)
	}

	// record event with the EventRecorder
	for _, r := range p.recorder {
		return handled, r.RecordEvent(id, ev, d)
	}

	return handled, nil
}
