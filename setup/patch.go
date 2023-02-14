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
	"github.com/jetsetilly/gopher2600/patch"
)

const patchEntryType = "patch"

const (
	patchFieldCartHash int = iota
	patchFieldCartName
	patchFieldNotes
	numPatchFields
)

// Patch is used to patch cartridge memory after cartridge has been
// attached/loaded.
type Patch struct {
	cartHash string
	cartName string
	notes    string
}

func deserialisePatchEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &Patch{}

	// basic sanity check
	if len(fields) > numPatchFields {
		return nil, fmt.Errorf("patch: too many fields in patch entry")
	}
	if len(fields) < numPatchFields {
		return nil, fmt.Errorf("patch: too few fields in patch entry")
	}

	set.cartHash = fields[patchFieldCartHash]
	set.cartName = fields[patchFieldCartName]
	set.notes = fields[patchFieldNotes]

	return set, nil
}

// EntryType implements the database.Entry interface.
func (set Patch) EntryType() string {
	return patchEntryType
}

// Serialise implements the database.Entry interface.
func (set *Patch) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
		set.cartHash,
		set.cartName,
		set.notes,
	}, nil
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
func (set Patch) apply(vcs *hardware.VCS) (string, error) {
	_, err := patch.CartridgeMemory(vcs.Mem.Cart, set.cartHash)
	if err != nil {
		return "", fmt.Errorf("patch: %w", err)
	}
	return fmt.Sprintf("patching cartridge: %s: %s", set.cartName, set.notes), nil
}
