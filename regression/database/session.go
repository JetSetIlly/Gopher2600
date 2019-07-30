package database

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Session keeps track of a database session
type Session struct {
	dbfile *os.File

	entries map[int]Entry

	// sorted list of keys. used for:
	// - displaying entries in correct order in list()
	// - saving in correct order in endSession()
	keys []int

	entryTypes map[string]deserialiser
}

// StartSession starts/initialises a new DB session. argument is the function to call when
// database has been succesfully opened
func StartSession(path string, init func(*Session) error) (*Session, error) {
	var err error

	db := &Session{}
	db.entryTypes = make(map[string]deserialiser)

	db.dbfile, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
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

// EndSession closes the database
func (db *Session) EndSession(commitChanges bool) error {
	// write entries to database
	if commitChanges {
		err := db.dbfile.Truncate(0)
		if err != nil {
			return err
		}

		_, err = db.dbfile.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}

		for _, key := range db.keys {
			s := strings.Builder{}
			ser, err := db.entries[key].Serialise()
			if err != nil {
				return err
			}

			s.WriteString(recordHeader(db.entries[key]))

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

func (db *Session) readDBFile() error {
	// clobbers the contents of db.entries
	db.entries = make(map[int]Entry, len(db.entries))

	// make sure we're at the beginning of the file
	if _, err := db.dbfile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	buffer, err := ioutil.ReadAll(db.dbfile)
	if err != nil {
		return errors.NewFormattedError(errors.RegressionDBError, err)
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
			msg := fmt.Sprintf("invalid key [%s] at line %d", fields[leaderFieldKey], i+1)
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		if _, ok := db.entries[key]; ok {
			msg := fmt.Sprintf("duplicate key [%v] at line %d", key, i+1)
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		var ent Entry

		init, ok := db.entryTypes[fields[leaderFieldID]]
		if !ok {
			msg := fmt.Sprintf("unrecognised entry type [%s]", fields[leaderFieldID])
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}
		ent, err = init(key, fields[numLeaderFields])
		if err != nil {
			return err
		}

		db.entries[key] = ent

		// add key to list
		db.keys = append(db.keys, key)
	}

	// sort key list
	sort.Ints(db.keys)

	return nil
}
