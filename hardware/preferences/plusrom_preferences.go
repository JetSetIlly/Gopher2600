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
	"fmt"
	"math/rand"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom/plusnet"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/resources"
)

type PlusROMPreferences struct {
	dsk *prefs.Disk

	Nick prefs.String
	ID   prefs.String

	// is true if the default nick/id are being used
	NewInstallation bool
}

func newPlusROMpreferences() (*PlusROMPreferences, error) {
	p := &PlusROMPreferences{}
	p.SetDefaults()

	pth, err := resources.JoinPath(prefs.DefaultPrefsFile)
	if err != nil {
		return nil, err
	}

	p.dsk, err = prefs.NewDisk(pth)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("plusrom.nick", &p.Nick)
	if err != nil {
		return nil, err
	}

	err = p.dsk.Add("plusrom.id", &p.ID)
	if err != nil {
		return nil, err
	}

	err = p.Nick.Set("gopher2600")
	if err != nil {
		return nil, err
	}

	err = p.ID.Set(fmt.Sprintf("%d", rand.Int63()))
	if err != nil {
		return nil, err
	}

	p.NewInstallation, err = p.dsk.HasEntry("plusrom.nick")
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load(false)
	if err != nil {
		return p, err
	}

	return p, nil
}

// SetDefaults reverts all settings to default values.
func (p *PlusROMPreferences) SetDefaults() {
	p.Nick.SetMaxLen(plusnet.MaxNickLength)
	p.ID.SetMaxLen(plusnet.MaxIDLength)
}

// Load plusrom preferences from disk.
func (p *PlusROMPreferences) Load() error {
	return p.dsk.Load(false)
}

// Save current plusrom preferences to disk.
func (p *PlusROMPreferences) Save() error {
	return p.dsk.Save()
}
