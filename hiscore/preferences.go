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

package hiscore

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

type Preferences struct {
	dsk *prefs.Disk

	AuthToken prefs.String
	Server    prefs.String
}

func (p Preferences) String() string {
	return p.dsk.String()
}

// newPreferences is the preferred method of initialisation for the Preferences type
func newPreferences() (*Preferences, error) {
	p := &Preferences{}

	// save server using the prefs package
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, errors.New(errors.HiScore, err)
	}

	p.dsk, err = prefs.NewDisk(pth)
	p.dsk.Add("hiscore.server", &p.Server)
	p.dsk.Add("hiscore.authtoken", &p.AuthToken)

	err = p.dsk.Load()
	if err != nil {
		return p, errors.New(errors.HiScore, err)
	}

	return p, nil
}

// Load hiscore preferences from disk
func (p *Preferences) Load() error {
	return p.dsk.Load()
}

// Save current hiscore preferences to disk
func (p *Preferences) Save() error {
	return p.dsk.Save()
}
