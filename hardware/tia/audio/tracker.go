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

package audio

import "github.com/jetsetilly/gopher2600/environment"

// TrackerEnvironment defines the subset of the Environment type required
// by a Tracker implementation
type TrackerEnvironment interface {
	AllowLogging() bool
	IsEmulation(environment.Label) bool
}

// Tracker implementations display or otherwise record the state of the audio registers for each channel
type Tracker interface {
	AUDCx(env TrackerEnvironment, channel int, data uint8)
	AUDFx(env TrackerEnvironment, channel int, data uint8)
	AUDVx(env TrackerEnvironment, channel int, data uint8)
}

// stub implementation of the tracker interface
type trackerStub struct{}

func (_ trackerStub) AUDCx(env TrackerEnvironment, channel int, data uint8) {
}

func (_ trackerStub) AUDFx(env TrackerEnvironment, channel int, data uint8) {
}

func (_ trackerStub) AUDVx(env TrackerEnvironment, channel int, data uint8) {
}
