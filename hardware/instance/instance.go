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

// Package instance defines those parts of the emulation that might change from
// instance to instance of the VCS type, but is not actually the VCS itself.
//
// Particularly useful when running more than one instance of the emulation in
// parallel.
package instance

import (
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/random"
)

// Instance defines those parts of the emulation that might change between
// different instantiations of the VCS type, but is not actually the VCS
// itself.
type Instance struct {
	Prefs  *preferences.Preferences
	Random *random.Random
}

// NewInstance is the preferred method of initialisation for the Instance type.
func NewInstance(coords signal.TelevisionCoords) (*Instance, error) {
	ins := &Instance{
		Random: random.NewRandom(coords),
	}

	var err error

	ins.Prefs, err = preferences.NewPreferences()
	if err != nil {
		return nil, err
	}

	return ins, nil
}

// Normalise ensures the VCS instance is in an known default state. Useful for
// regression testing where the initial state must be the same for every run of
// the test.
func (ins *Instance) Normalise() {
	ins.Random.ZeroSeed = true
	ins.Prefs.SetDefaults()
	ins.Prefs.Revision.SetDefaults()
	ins.Prefs.ARM.SetDefaults()
	ins.Prefs.PlusROM.SetDefaults()
}
