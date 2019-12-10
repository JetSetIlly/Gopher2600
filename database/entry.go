package database

import (
	"fmt"
	"gopher2600/errors"
)

// Deserialiser extracts/converts fields from a SerialisedEntry
type Deserialiser func(fields SerialisedEntry) (Entry, error)

// SerialisedEntry is the Entry data represented as an array of strings
type SerialisedEntry []string

// Entry represents the generic entry in the database
type Entry interface {
	// ID returns the string that is used to identify the entry type in
	// the database
	ID() string

	// String should return information about the entry in a human readable
	// format. by contrast, machine readable representation is returned by the
	// Serialise function
	String() string

	// return the Entry data as an instance of SerialisedEntry.
	Serialise() (SerialisedEntry, error)

	// a clenaup is perfomed when entry is deleted from the database
	CleanUp() error
}

// RegisterEntryType tells the database what entries it may expect in the database
// and how to deserialise the entry.
func (db *Session) RegisterEntryType(id string, des Deserialiser) error {
	if _, ok := db.entryTypes[id]; ok {
		msg := fmt.Sprintf("trying to register a duplicate entry ID [%s]", id)
		return errors.New(errors.DatabaseError, msg)
	}
	db.entryTypes[id] = des
	return nil
}
