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

package instance

import (
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/random"
)

// Label indicates the context of the instance.
type Label string

// List of value Label values.
const (
	Main        Label = ""
	Thumbnailer Label = "thumbnailer"
	Comparison  Label = "comparison"
)

// Instance defines those parts of the emulation that might change between
// different instantiations of the VCS type, but is not actually the VCS
// itself.
type Instance struct {
	// the name of the instance. the name can be used to decide if an action is appropriate
	Label Label

	// any randomisation required by the emulation should be retreived through
	// this structure
	Random *random.Random

	// the prefrences of the running instance. this instance can be shared
	// with other running instances of the emulation.
	Prefs *preferences.Preferences
}

// NewInstance is the preferred method of initialisation for the Instance type.
//
// The two arguments must be supplied. In the case of the prefs field it can by
// nil and a new prefs instance will be created. Providing a non-nil value
// allows the preferences of more than one VCS instance to be synchronised.
func NewInstance(tv random.TV, prefs *preferences.Preferences) (*Instance, error) {
	ins := &Instance{
		Random: random.NewRandom(tv),
	}

	var err error

	if prefs == nil {
		prefs, err = preferences.NewPreferences()
		if err != nil {
			return nil, err
		}
	}

	ins.Prefs = prefs

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
