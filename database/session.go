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
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
)

// Sentinal error returned when requested database is not available.
const (
	NotAvailable = "database: not available (%s)"
)

// Activity is used to specify the general activity of what will be occurring
// during the database session.
type Activity int

// Valid activities: the "higher level" activities inherit the activity
// abilities of the activity levels lower down the scale.
const (
	ActivityReading Activity = iota

	// Modifying implies Reading.
	ActivityModifying

	// Creating implies Modifying (which in turn implies Reading).
	ActivityCreating
)

// Session keeps track of a database session.
type Session struct {
	dbfile   *os.File
	activity Activity

	entries map[int]Entry

	// deserialisers for the different entries that may appear in the database
	entryTypes map[string]Deserialiser
}

// StartSession starts/initialises a new DB session. argument is the function
// to call when database has been successfully opened. this function should be
// used to add information about the different entries that are to be used in
// the database (see AddEntryType() function).
func StartSession(path string, activity Activity, init func(*Session) error) (*Session, error) {
	var err error

	db := &Session{activity: activity}
	db.entryTypes = make(map[string]Deserialiser)

	var flags int
	switch activity {
	case ActivityReading:
		flags = os.O_RDONLY
	case ActivityModifying:
		flags = os.O_RDWR
	case ActivityCreating:
		flags = os.O_RDWR | os.O_CREATE
	}

	db.dbfile, err = os.OpenFile(path, flags, 0600)
	if err != nil {
		switch err.(type) {
		case *os.PathError:
			return nil, curated.Errorf(NotAvailable, path)
		}
		return nil, curated.Errorf("databas: %v", err)
	}

	// closing of db.dbfile requires a call to endSession()

	err = init(db)
	if err != nil {
		return nil, err
	}

	err = db.readDBFile()
	if err != nil {
		return nil, err
	}

	return db, nil
}

// EndSession closes the database.
func (db *Session) EndSession(commitChanges bool) error {
	// write entries to database
	if commitChanges {
		if db.activity == ActivityReading {
			return curated.Errorf("database: cannot commit to a read-only database")
		}

		err := db.dbfile.Truncate(0)
		if err != nil {
			return err
		}

		_, err = db.dbfile.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}

		for k, v := range db.entries {
			s := strings.Builder{}
			ser, err := v.Serialise()
			if err != nil {
				return err
			}

			s.WriteString(recordHeader(k, v.ID()))

			for i := 0; i < len(ser); i++ {
				s.WriteString(fieldSep)
				s.WriteString(ser[i])
			}

			s.WriteString(entrySep)

			_, err = db.dbfile.WriteString(s.String())
			if err != nil {
				return err
			}
		}
	}

	// end session by closing file
	if db.dbfile != nil {
		err := db.dbfile.Close()
		if err != nil {
			return err
		}
		db.dbfile = nil
	}

	return nil
}

// readDBFile reads each line in the database file, checks for validity of key
// and entry type and tries to deserialise the entry. it fails on the first
// error it encounters.
func (db *Session) readDBFile() error {
	// clobbers the contents of db.entries
	db.entries = make(map[int]Entry, len(db.entries))

	// make sure we're at the beginning of the file
	if _, err := db.dbfile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	buffer, err := ioutil.ReadAll(db.dbfile)
	if err != nil {
		return curated.Errorf("database: %v", err)
	}

	// split entries
	lines := strings.Split(string(buffer), entrySep)

	for i := 0; i < len(lines); i++ {
		lines[i] = strings.TrimSpace(lines[i])
		if len(lines[i]) == 0 {
			continue
		}

		// loop through file until EOF is reached
		fields := strings.SplitN(lines[i], fieldSep, numLeaderFields+1)

		key, err := strconv.Atoi(fields[leaderFieldKey])
		if err != nil {
			return curated.Errorf("invalid key (%s) [line %d]", fields[leaderFieldKey], i+1)
		}

		if _, ok := db.entries[key]; ok {
			return curated.Errorf("duplicate key (%s) [line %d]", key, i+1)
		}

		var ent Entry

		deserialise, ok := db.entryTypes[fields[leaderFieldID]]
		if !ok {
			return curated.Errorf("unrecognised entry type (%s) [line %d]", fields[leaderFieldID], i+1)
		}

		ent, err = deserialise(strings.Split(fields[numLeaderFields], ","))
		if err != nil {
			return curated.Errorf("%v [line %d]", err, i+1)
		}

		db.entries[key] = ent
	}

	return nil
}
