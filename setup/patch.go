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
	"github.com/jetsetilly/gopher2600/patch"
)

const patchID = "patch"

const (
	patchFieldCartHash int = iota
	patchFieldPatchFile
	patchFieldNotes
	numPatchFields
)

// Patch is used to patch cartridge memory after cartridge has been
// attached/loaded.
type Patch struct {
	cartHash  string
	patchFile string
	notes     string
}

func deserialisePatchEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &Patch{}

	// basic sanity check
	if len(fields) > numPatchFields {
		return nil, curated.Errorf("patch: too many fields in patch entry")
	}
	if len(fields) < numPatchFields {
		return nil, curated.Errorf("patch: too few fields in patch entry")
	}

	set.cartHash = fields[patchFieldCartHash]
	set.patchFile = fields[patchFieldPatchFile]
	set.notes = fields[patchFieldNotes]

	return set, nil
}

// ID implements the database.Entry interface.
func (set Patch) ID() string {
	return patchID
}

// String implements the database.Entry interface.
func (set Patch) String() string {
	return fmt.Sprintf("%s, %s", set.cartHash, set.patchFile)
}

// Serialise implements the database.Entry interface.
func (set *Patch) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			set.cartHash,
			set.patchFile,
			set.notes,
		},
		nil
}

// CleanUp implements the database.Entry interface.
func (set Patch) CleanUp() error {
	// no cleanup necessary
	return nil
}

// matchCartHash implements setupEntry interface.
func (set Patch) matchCartHash(hash string) bool {
	return set.cartHash == hash
}

// apply implements setupEntry interface.
func (set Patch) apply(vcs *hardware.VCS) error {
	_, err := patch.CartridgeMemory(vcs.Mem.Cart, set.patchFile)
	if err != nil {
		return curated.Errorf("patch: %v", err)
	}
	return nil
}
