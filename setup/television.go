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

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/hardware"
)

const televisionID = "television"

const (
	televisionFieldCartHash int = iota
	televisionFieldSpec
	televisionFieldNotes
	numtelevisionFields
)

// television is used to television cartridge memory after cartridge has been
// attached/loaded.
type television struct {
	cartHash string
	spec     string
	notes    string
}

func deserialiseTelevisionEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &television{}

	// basic sanity check
	if len(fields) > numtelevisionFields {
		return nil, curated.Errorf("television: too many fields in television entry")
	}
	if len(fields) < numtelevisionFields {
		return nil, curated.Errorf("television: too few fields in television entry")
	}

	set.cartHash = fields[televisionFieldCartHash]
	set.spec = fields[televisionFieldSpec]
	set.notes = fields[televisionFieldNotes]

	return set, nil
}

// ID implements the database.Entry interface.
func (set television) ID() string {
	return televisionID
}

// String implements the database.Entry interface.
func (set television) String() string {
	return fmt.Sprintf("%s, %s", set.cartHash, set.spec)
}

// Serialise implements the database.Entry interface.
func (set *television) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			set.cartHash,
			set.spec,
			set.notes,
		},
		nil
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
func (set television) apply(vcs *hardware.VCS) error {
	return vcs.TV.SetSpec(set.spec)
}
