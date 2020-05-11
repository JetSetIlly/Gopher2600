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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package hiscore

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

type preferences struct {
	dsk       *prefs.Disk
	authToken prefs.String
	server    prefs.String
}

func loadPreferences() (*preferences, error) {
	p := &preferences{}

	// save server using the prefs package
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, errors.New(errors.HiScore, err)
	}

	p.dsk, err = prefs.NewDisk(pth)
	p.dsk.Add("hiscore.server", &p.server)
	p.dsk.Add("hiscore.authtoken", &p.authToken)

	err = p.dsk.Load()
	if err != nil {
		return p, errors.New(errors.HiScore, err)
	}

	return p, nil
}

func (p *preferences) save() error {
	return p.dsk.Save()
}
