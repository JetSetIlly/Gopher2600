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
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/hardware/memory/bus"
)

// ID differentiates the different ports attached to the console
type ID int

// List of defined IDs
const (
	PlayerZeroID ID = iota
	PlayerOneID
	PanelID
	NumIDs
)

// Peripheral represents a (input or output) device that can attached to the
// VCS ports.
type Peripheral interface {
	String() string
	Reset()

	Update(bus.ChipData) bool
	Step()

	// HandleEvent sends an event and data to the port. If the Peripheral
	// attached to the port does not handle that type of Event it should simply
	// probably just the event silently
	HandleEvent(Event, EventData) error
}

// PeripheralConstructor defines the function signature for a creating a new
// peripheral, suitable for use with AttachPloyer0() and AttachPlayer1()
type PeripheralConstructor func(*MemoryAccess) Peripheral

// RecordablePeripheral defines the function a peripheral has if it is the
// ability to be recorded or played back
type RecordablePeripheral interface {
	AttachPlayback(EventPlayback)
	AttachEventRecorder(EventRecorder)
	GetPlayback() error
}

// Recordable implements the basic functionality for a RecordablePort
// implementation. Embed the struct in a RecordablePeripheral implementation.
type Recordable struct {
	ID          ID
	Playback    EventPlayback
	Recorder    EventRecorder
	HandleEvent func(Event, EventData) error
}

// GetPlayback implements the Recordable interface. Checks for a playback
// event and handles it.
func (p *Recordable) GetPlayback() error {
	if p.Playback != nil {
		ev, v, err := p.Playback.GetPlaybackEvent(p.ID)
		if err != nil {
			return err
		}

		err = p.HandleEvent(ev, v)
		if err != nil {
			if !errors.Is(err, errors.InputDeviceUnplugged) {
				return err
			}
			p.AttachPlayback(nil)
		}
	}

	return nil
}

// AttachPlayback implements the Recordable interface. Attaches an
// EventPlayback implementation to the port. An EventPlayback value of nil will
// remove the playback from the port
func (p *Recordable) AttachPlayback(playback EventPlayback) {
	p.Playback = playback
}

// AttachEventRecorder implements the Recordable interface. An EventRecorder
// value of nil will remove the recorder from the port.
func (p *Recordable) AttachEventRecorder(recorder EventRecorder) {
	p.Recorder = recorder
}
