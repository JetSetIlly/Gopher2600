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

package database

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/curated"
)

// Deserialiser extracts/converts fields from a SerialisedEntry.
type Deserialiser func(fields SerialisedEntry) (Entry, error)

// SerialisedEntry is the Entry data represented as an array of strings.
type SerialisedEntry []string

// Entry represents the generic entry in the database.
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
		return curated.Errorf("database: %v", msg)
	}
	db.entryTypes[id] = des
	return nil
}
