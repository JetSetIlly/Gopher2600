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

// Package input represents the input/output part of the RIOT (the IO in RIOT).
// Note that the output aspect of the RIOT is minimal and so the package is
// appropriately called input.
//
// The main type in the package, the Input type, contains references to the
// three input devices - the panel and the two hand controller ports.
//
// The HandController type handles the input from all types of hand controllers
// (although, currently, joystick and paddle only)
//
// The Panel type handles the input from the VCS's front panel switches.
//
// The Panel and HandController types satisfy the Port type.
//
// Physical controllers for the emulation can interact with the Panel and
// HandController types throught the Handle() function and pass the correct
// Event to indicate the desired effect.
//
// An alternative to the Handle() function is the Playback interface. The Port
// interface can have a Playback instance attached to it with the
// AttachPlayback() function. The CheckInput function of the Playback interface
// can then be used to check for Events.
//
// The Playback interface is intended as the counterpart to the EventRecorder
// interface, but it could theoretically be used in other contexts.
//
// EventRecorders intercept all events issued to either of the Port
// implementations and handle those events in their own way. The intended
// purpose is for the events to be recorded to disk for future playback. Note
// that events issued by Playback implementations are also passed through
// attached Event Recorders.
package input
