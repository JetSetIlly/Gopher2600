package database

import (
	"fmt"
	"gopher2600/errors"
)

// Key type is only used within the Entry type. it is intended to obfuscate the
// key's real type and to discourage the Entry implementation from altering the
// key in any way
type Key struct {
	hiddenKey int
}

// the initialisation function when creating a new entry
type deserialiser func(key Key, fields []string) (Entry, error)

// SerialisedEntry is the Entry data represented as an array of strings
type SerialisedEntry []string

// Entry represents the generic entry in the database
type Entry interface {
	// String implements the Stringer interface
	String() string

	// getID returns the string that is used to identify the entry type in
	// the database
	GetID() string

	// set the key value for the entry
	SetKey(Key)

	// return the key assigned to the entry
	GetKey() Key

	// return the Entry data as an instance of SerialisedEntry
	Serialise() (SerialisedEntry, error)

	// action perfomed when entry is removed from database
	CleanUp()
}

// AddEntryType tells the database what entries it may expect in the database
// and what to do when it encounters one
func (db *Session) AddEntryType(id string, des deserialiser) error {
	if _, ok := db.entryTypes[id]; ok {
		msg := fmt.Sprintf("trying to register a duplicate entry ID [%s]", id)
		return errors.New(errors.DatabaseError, msg)
	}
	db.entryTypes[id] = des
	return nil
}
