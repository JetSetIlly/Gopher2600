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
	"io"
	"sort"

	"github.com/jetsetilly/gopher2600/curated"
)

// arbitrary maximum number of entries.
const maxEntries = 1000

const fieldSep = ","
const entrySep = "\n"

const (
	leaderFieldKey int = iota
	leaderFieldID
	numLeaderFields
)

func recordHeader(key int, id string) string {
	return fmt.Sprintf("%03d%s%s", key, fieldSep, id)
}

// NumEntries returns the number of entries in the database.
func (db Session) NumEntries() int {
	return len(db.entries)
}

// SortedKeyList returns a sorted list of database keys.
func (db Session) SortedKeyList() []int {
	// sort entries into key order
	keyList := make([]int, 0, len(db.entries))
	for k := range db.entries {
		keyList = append(keyList, k)
	}
	sort.Ints(keyList)
	return keyList
}

// List the enties in key order.
func (db Session) List(output io.Writer) error {
	if db.NumEntries() == 0 {
		if _, err := output.Write([]byte("database is empty\n")); err != nil {
			return err
		}
		return nil
	}

	keyList := db.SortedKeyList()

	for k := range keyList {
		key := keyList[k]
		ent := db.entries[key]

		if _, err := output.Write([]byte(fmt.Sprintf("%03d ", key))); err != nil {
			return err
		}

		if _, err := output.Write([]byte(ent.String())); err != nil {
			return err
		}
		if _, err := output.Write([]byte("\n")); err != nil {
			return err
		}
	}

	if _, err := output.Write([]byte(fmt.Sprintf("Total: %d\n", db.NumEntries()))); err != nil {
		return err
	}

	return nil
}

// Add an entry to the db.
func (db *Session) Add(ent Entry) error {
	var key int

	// find spare key
	for key = 0; key < maxEntries; key++ {
		if _, ok := db.entries[key]; !ok {
			break
		}
	}

	if key == maxEntries {
		return curated.Errorf("database: maximum entries exceeded (max %d)", maxEntries)
	}

	db.entries[key] = ent

	return nil
}

// Delete deletes an entry with the specified key. returns DatabaseKeyError
// if not such entry exists.
func (db *Session) Delete(key int) error {
	ent, ok := db.entries[key]
	if !ok {
		return curated.Errorf("database: key not available (%s)", key)
	}

	if err := ent.CleanUp(); err != nil {
		return curated.Errorf("database: %v", err)
	}

	delete(db.entries, key)

	return nil
}
