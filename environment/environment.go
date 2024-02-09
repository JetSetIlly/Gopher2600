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

package environment

import (
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/random"
)

// Label is used to name the environment
type Label string

// Television interface exposing a minimum amount of the real television
// implementation
type Television interface {
	GetSpecID() string
	SetRotation(specification.Rotation)
}

// Environment is used to provide context for an emulation. Particularly useful
// when using multiple emulations
type Environment struct {
	// label distinguishes between different types of emulation (thumbnailer, etc.)
	Label Label

	// the television attached to the console
	TV Television

	// any randomisation required by the emulation should be retreived through
	// this structure
	Random *random.Random

	// the emulation preferences
	Prefs *preferences.Preferences
}

// NewEnvironment is the preferred method of initialisation for the Environment type.
//
// The two arguments must be supplied. In the case of the prefs field it can by
// nil and a new Preferences instance will be created. Providing a non-nil value
// allows the preferences of more than one VCS emulation to be synchronised.
func NewEnvironment(tv *television.Television, prefs *preferences.Preferences) (*Environment, error) {
	env := &Environment{
		TV:     tv,
		Random: random.NewRandom(tv),
	}

	var err error

	if prefs == nil {
		prefs, err = preferences.NewPreferences()
		if err != nil {
			return nil, err
		}
	}

	env.Prefs = prefs

	return env, nil
}

// Normalise ensures the environment is in an known default state. Useful for
// regression testing where the initial state must be the same for every run of
// the test.
func (env *Environment) Normalise() {
	env.Random.ZeroSeed = true
	env.Prefs.SetDefaults()
	env.Prefs.Revision.SetDefaults()
	env.Prefs.ARM.SetDefaults()
	env.Prefs.PlusROM.SetDefaults()
}

// MainEmulation is the label used for the main emulation
const MainEmulation = Label("main")

// IsEmulation checks the emulation label and returns true if it matches
func (env *Environment) IsEmulation(label Label) bool {
	return env.Label == label
}
