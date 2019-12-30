package setup

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/errors"
	"gopher2600/hardware"
)

const televisionID = "television"

const (
	televisionFieldCartHash int = iota
	televisionFieldSpec
	televisionFieldNotes
	numtelevisionFields
)

// television is used to television cartridge memory after cartridge has been
// attached/loaded
type television struct {
	cartHash string
	spec     string
	notes    string
}

func deserialiseTelevisionEntry(fields database.SerialisedEntry) (database.Entry, error) {
	set := &television{}

	// basic sanity check
	if len(fields) > numtelevisionFields {
		return nil, errors.New(errors.SetupTelevisionError, "too many fields in television entry")
	}
	if len(fields) < numtelevisionFields {
		return nil, errors.New(errors.SetupTelevisionError, "too few fields in television entry")
	}

	set.cartHash = fields[televisionFieldCartHash]
	set.spec = fields[televisionFieldSpec]
	set.notes = fields[televisionFieldNotes]

	return set, nil
}

// ID implements the database.Entry interface
func (set television) ID() string {
	return televisionID
}

// String implements the database.Entry interface
func (set television) String() string {
	return fmt.Sprintf("%s, %s", set.cartHash, set.spec)
}

// Serialise implements the database.Entry interface
func (set *television) Serialise() (database.SerialisedEntry, error) {
	return database.SerialisedEntry{
			set.cartHash,
			set.spec,
			set.notes,
		},
		nil
}

// CleanUp implements the database.Entry interface
func (set television) CleanUp() error {
	// no cleanup necessary
	return nil
}

// matchCartHash implements setupEntry interface
func (set television) matchCartHash(hash string) bool {
	return set.cartHash == hash
}

// apply implements setupEntry interface
func (set television) apply(vcs *hardware.VCS) error {
	return vcs.TV.SetSpec(set.spec)
}
