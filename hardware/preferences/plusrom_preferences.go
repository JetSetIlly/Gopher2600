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
	"crypto/md5"
	"fmt"
	"math/rand"
	"unicode"

	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom/plusnet"
	"github.com/jetsetilly/gopher2600/logger"
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

	err = p.dsk.Add("plusrom.id_v2.1.1", &p.ID)
	if err != nil {
		return nil, err
	}

	err = p.Nick.Set("gopher2600")
	if err != nil {
		return nil, err
	}

	err = p.ID.Set(p.generateID())
	if err != nil {
		return nil, err
	}

	p.NewInstallation, err = p.dsk.DoesNotHaveEntry("plusrom.nick")
	if err != nil {
		return nil, err
	}

	err = p.dsk.Load(true)
	if err != nil {
		return p, err
	}

	if !p.validateID() {
		logger.Log("plusrom preferences", "existing ID invalid. generating new ID and saving")

		err = p.ID.Set(p.generateID())
		if err != nil {
			return nil, err
		}

		err = p.dsk.Save()
		if err != nil {
			return p, err
		}
	}

	return p, nil
}

func (p *PlusROMPreferences) validateID() bool {
	// check for length
	if len(p.ID.String()) != plusnet.MaxIDLength {
		return false
	}

	// check for invalid characters
	for _, r := range p.ID.String() {
		if !((unicode.IsLetter(r) && unicode.IsLower(r)) || unicode.IsDigit(r)) {
			return false
		}
	}

	return true
}

func (p *PlusROMPreferences) generateID() string {
	id := fmt.Sprintf("%d", rand.Int63())
	id = fmt.Sprintf("%x", md5.Sum([]byte(id)))
	return id
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
