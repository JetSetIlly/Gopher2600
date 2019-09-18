package database

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"sort"
)

// arbitrary maximum number of entries
const maxEntries = 1000

const fieldSep = ","
const entrySep = "\n"

const (
	leaderFieldKey int = iota
	leaderFieldID
	numLeaderFields
)

func recordHeader(ent Entry) string {
	return fmt.Sprintf("%03d%s%s", ent.GetKey(), fieldSep, ent.GetID())
}

// NumEntries returns the number of entries in the database
func (db Session) NumEntries() int {
	return len(db.keys)
}

// List the enties in key order
func (db Session) List(output io.Writer) error {
	for k := range db.keys {
		if _, err := output.Write([]byte(fmt.Sprintf("%03d ", db.keys[k]))); err != nil {
			return err
		}
		if _, err := output.Write([]byte(db.entries[db.keys[k]].String())); err != nil {
			return err
		}
		if _, err := output.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if len(db.keys) == 0 {
		if _, err := output.Write([]byte("database is empty\n")); err != nil {
			return err
		}
	} else {
		if _, err := output.Write([]byte(fmt.Sprintf("Total: %d\n", len(db.keys)))); err != nil {
			return err
		}
	}
	return nil
}

// Add an entry to the db
func (db *Session) Add(ent Entry) error {
	var key int

	// find spare key
	for key = 0; key < maxEntries; key++ {
		if _, ok := db.entries[key]; !ok {
			break
		}
	}

	if key == maxEntries {
		msg := fmt.Sprintf("%d maximum entries exceeded", maxEntries)
		return errors.New(errors.DatabaseError, msg)
	}

	ent.SetKey(key)
	db.entries[key] = ent

	// add key to list and resort
	db.keys = append(db.keys, key)
	sort.Ints(db.keys)

	return nil
}

// Delete an entry from the database
func (db *Session) Delete(ent Entry) error {
	ent.CleanUp()

	delete(db.entries, ent.GetKey())

	// find key in list and delete
	for i := 0; i < len(db.keys); i++ {
		if db.keys[i] == ent.GetKey() {
			db.keys = append(db.keys[:i], db.keys[i+1:]...)
			break // for loop
		}
	}

	return nil
}

// Get an entry from the database
func (db Session) Get(key int) (Entry, error) {
	ent, ok := db.entries[key]
	if !ok {
		msg := fmt.Sprintf("key not found [%d]", key)
		return nil, errors.New(errors.DatabaseError, msg)
	}
	return ent, nil
}

// Select entries based on match calling onSelect for each matched entry
// !!TODO: match not implemented yet. function will match on every entry
func (db Session) Select(match string, onSelect func(Entry) (bool, error)) error {
	for i := range db.keys {
		cont, err := onSelect(db.entries[db.keys[i]])
		if err != nil {
			return err
		}
		if !cont {
			break // for loop
		}
	}

	return nil
}
