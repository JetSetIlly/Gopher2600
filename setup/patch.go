package setup

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/patch"
)

const patchID = "patch"

const (
	patchFieldCartHash int = iota
	patchFieldPatchFile
	patchFieldNotes
	numPatchFields
)

// Patch is used to patch cartridge memory after cartridge has been
// attached/loaded
type Patch struct {
	cartHash  string
	patchFile string
	notes     string
}

func deserialisePatchEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &Patch{}

	// basic sanity check
	if len(fields) > numPatchFields {
		return nil, errors.New(errors.SetupPatchError, "too many fields in patch entry")
	}
	if len(fields) < numPatchFields {
		return nil, errors.New(errors.SetupPatchError, "too few fields in patch entry")
	}

	set.cartHash = fields[patchFieldCartHash]
	set.patchFile = fields[patchFieldPatchFile]
	set.notes = fields[patchFieldNotes]

	return set, nil
}

// ID implements the database.Entry interface
func (set Patch) ID() string {
	return patchID
}

// String implements the database.Entry interface
func (set Patch) String() string {
	return fmt.Sprintf("%s, %s", set.cartHash, set.patchFile)
}

// Serialise implements the database.Entry interface
func (set *Patch) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			set.cartHash,
			set.patchFile,
			set.notes,
		},
		nil
}

// CleanUp implements the database.Entry interface
func (set Patch) CleanUp() error {
	// no cleanup necessary
	return nil
}

// matchCartHash implements setupEntry interface
func (set Patch) matchCartHash(hash string) bool {
	return set.cartHash == hash
}

// apply implements setupEntry interface
func (set Patch) apply(vcs *hardware.VCS) error {
	_, err := patch.CartridgeMemory(vcs.Mem.Cart, set.patchFile)
	if err != nil {
		return errors.New(errors.SetupPatchError, err)
	}
	return nil
}
