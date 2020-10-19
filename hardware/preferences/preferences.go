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

package preferences

import (
	"math/rand"
	"time"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

// Preferences defines and collates all the preference values used by the debugger.
type Preferences struct {
	dsk *prefs.Disk

	// initialise hardware to unknown state after reset
	RandomState prefs.Bool

	// unused pins when reading TIA/RIOT registers take the value of the last
	// value on the bus. if RandomPins is true then the values of the unusued
	// pins are randomised. this is the equivalent of the Stella option "drive
	// unused pins randomly on a read/peek"
	RandomPins prefs.Bool

	// random values generated in the hardware package should use the following
	// number source
	RandSrc *rand.Rand

	// the number used to seed RandSrc
	RandSeed int64
}

func (p *Preferences) String() string {
	return p.dsk.String()
}

// NewPreferences is the preferred method of initialisation for the Preferences type.
func NewPreferences() (*Preferences, error) {
	p := &Preferences{}

	// initialise random number generator
	p.Reseed(0)

	// setup preferences and load from disk
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.randstate", &p.RandomState)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("hardware.randpins", &p.RandomPins)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Load(true)
	if err != nil {
		// ignore missing prefs file errors
		if !curated.Is(err, prefs.NoPrefsFile) {
			return nil, err
		}
	}

	return p, nil
}

// Reseed initialises the random number generator. Use a seed value of 0 to
// initialise with the current time.
func (p *Preferences) Reseed(seed int64) {
	if seed == 0 {
		p.RandSeed = int64(time.Now().Nanosecond())
	} else {
		p.RandSeed = seed
	}
	p.RandSrc = rand.New(rand.NewSource(p.RandSeed))
}

// Reset all hardware preferences to the default values.
func (p *Preferences) Reset() error {
	return p.dsk.Reset()
}

// Load current hardware preference from disk.
func (p *Preferences) Load() error {
	return p.dsk.Load(false)
}

// Save current hardware preferences to disk.
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
