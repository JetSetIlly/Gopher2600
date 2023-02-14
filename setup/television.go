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

package setup

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
)

const televisionEntryType = "tv"

const (
	televisionFieldCartHash int = iota
	televisionFieldCartName
	televisionFieldSpec
	numtelevisionFields
)

// television is used to television cartridge memory after cartridge has been
// attached/loaded.
type television struct {
	cartHash string
	cartName string
	spec     string
}

func deserialiseTelevisionEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &television{}

	// basic sanity check
	if len(fields) > numtelevisionFields {
		return nil, fmt.Errorf("television: too many fields in television entry")
	}
	if len(fields) < numtelevisionFields {
		return nil, fmt.Errorf("television: too few fields in television entry")
	}

	set.cartHash = fields[televisionFieldCartHash]
	set.cartName = fields[televisionFieldCartName]
	set.spec = fields[televisionFieldSpec]

	return set, nil
}

// EntryType implements the database.Entry interface.
func (set television) EntryType() string {
	return televisionEntryType
}

// Serialise implements the database.Entry interface.
func (set *television) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
		set.cartHash,
		set.cartName,
		set.spec,
	}, nil
}

// CleanUp implements the database.Entry interface.
func (set television) CleanUp() error {
	// no cleanup necessary
	return nil
}

// matchCartHash implements setupEntry interface.
func (set television) matchCartHash(hash string) bool {
	return set.cartHash == hash
}

// apply implements setupEntry interface.
func (set television) apply(vcs *hardware.VCS) (string, error) {
	// because the apply function is run after attaching the cartridge to the
	// VCS, any setup entries will take precedence over any spec in the
	// cartridge filename.
	//
	// the SetSpecConditional() function however, will only change spec if the
	// original spec request is AUTO. In other words, a setup entry will not
	// take precedence over an explicit startup option.

	err := vcs.TV.SetSpecConditional(set.spec)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("forcing %s mode: %s", set.spec, set.cartName), nil
}
