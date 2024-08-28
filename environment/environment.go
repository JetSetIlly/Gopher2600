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
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/hardware/preferences"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/gopher2600/random"
)

// Label is used to name the environment
type Label string

// MainEmulation is the label used for the main emulation
const MainEmulation = Label("main")

// Television interface exposing a minimum amount of the real television
// implementation
type Television interface {
	GetSpecID() string
	GetReqSpecID() string
	SetRotation(specification.Rotation)
}

// Environment is used to provide context for an emulation. Particularly useful
// when using multiple emulations
type Environment struct {
	// label distinguishes between different types of emulation (thumbnailer, etc.)
	Label Label

	// the television attached to the console
	TV Television

	// interface to emulation. used for example, when cartridge has been
	// successfully loaded. not all cartridge formats require this
	Notifications notifications.Notify

	// the emulation preferences
	Prefs *preferences.Preferences

	// any randomisation required by the emulation should be retreived through
	// this structure
	Random *random.Random

	// current cartridge loader
	Loader cartridgeloader.Loader
}

// NewEnvironment is the preferred method of initialisation for the Environment type.
//
// The Notify and Preferences can be nil. If prefs is nil then a new instance of
// the system wide preferences will be created.
func NewEnvironment(label Label, tv Television, notify notifications.Notify, prefs *preferences.Preferences) (*Environment, error) {
	env := &Environment{
		Label:         label,
		TV:            tv,
		Notifications: notify,
		Prefs:         prefs,
		Random:        random.NewRandom(tv.(random.TV)),
	}

	if notify == nil {
		env.Notifications = notificationStub{}
	}

	if prefs == nil {
		var err error
		env.Prefs, err = preferences.NewPreferences()
		if err != nil {
			return nil, err
		}
	}

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

// IsEmulation checks the emulation label and returns true if it matches
func (env *Environment) IsEmulation(label Label) bool {
	return env.Label == label
}

// AllowLogging returns true if environment is permitted to create new log entries
func (env *Environment) AllowLogging() bool {
	return env.IsEmulation(MainEmulation)
}

// stub implementation of the notification interface
type notificationStub struct{}

func (_ notificationStub) Notify(_ notifications.Notice) error {
	return nil
}
