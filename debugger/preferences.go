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

package debugger

import (
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/paths"
	"github.com/jetsetilly/gopher2600/prefs"
)

// Preferences defines and collates all the preference values used by the debugger
type Preferences struct {
	dbg *Debugger
	dsk *prefs.Disk

	RandomState *prefs.Bool
	RandomPins  *prefs.Bool
}

func (p Preferences) String() string {
	return p.dsk.String()
}

// newPreferences is the preferred method of initialisation for the Preferences type
func newPreferences(dbg *Debugger) (*Preferences, error) {
	p := &Preferences{
		dbg:         dbg,
		RandomState: &dbg.VCS.RandomState,
		RandomPins:  &dbg.VCS.Mem.RandomPins,
	}

	// setup preferences and load from disk
	pth, err := paths.ResourcePath("", prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}
	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("debugger.randstate", p.RandomState)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Add("debugger.randpins", p.RandomPins)
	if err != nil {
		return nil, err
	}
	err = p.dsk.Load(true)
	if err != nil {
		// ignore missing prefs file errors
		if !errors.Is(err, prefs.NoPrefsFile) {
			return nil, err
		}
	}

	return p, nil
}

func (p *Preferences) load() error {
	return p.dsk.Load(false)
}

func (p *Preferences) save() error {
	return p.dsk.Save()
}
