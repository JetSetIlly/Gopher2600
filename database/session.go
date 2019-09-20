package database

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// ActivityType is used to specify the general activity of what will be
// occuring during the database session
type ActivityType int

// a list of valid ActivityType. the "higher level" activities inherit the
// activity abilities of the activity levels lower down the scale. in other
// words:
//		- Modifying implies Reading
//		- Creating implies Modifying (which in turn implies Reading)
const (
	ActivityReading ActivityType = iota
	ActivityModifying
	ActivityCreating
)

// Session keeps track of a database session
type Session struct {
	dbfile   *os.File
	activity ActivityType

	entries map[int]Entry

	// deserialisers for the different entries that may appear in the database
	entryTypes map[string]deserialiser
}

// StartSession starts/initialises a new DB session. argument is the function
// to call when database has been succesfully opened. this function should be
// used to add information about the different entries that are to be used in
// the database (see AddEntryType() function)
func StartSession(path string, activity ActivityType, init func(*Session) error) (*Session, error) {
	var err error

	db := &Session{activity: activity}
	db.entryTypes = make(map[string]deserialiser)

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
			return nil, errors.New(errors.DatabaseFileUnavailable, path)
		}
		return nil, errors.New(errors.DatabaseError, err)
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
		if db.activity == ActivityReading {
			return errors.New(errors.DatabaseError, "cannot commit to a read-only database")
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

func (db *Session) readDBFile() error {
	// clobbers the contents of db.entries
	db.entries = make(map[int]Entry, len(db.entries))

	// make sure we're at the beginning of the file
	if _, err := db.dbfile.Seek(0, io.SeekStart); err != nil {
		return err
	}

	buffer, err := ioutil.ReadAll(db.dbfile)
	if err != nil {
		return errors.New(errors.DatabaseError, err)
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
			return errors.New(errors.DatabaseError, msg)
		}

		if _, ok := db.entries[key]; ok {
			msg := fmt.Sprintf("duplicate key [%v] at line %d", key, i+1)
			return errors.New(errors.DatabaseError, msg)
		}

		var ent Entry

		deserialise, ok := db.entryTypes[fields[leaderFieldID]]
		if !ok {
			msg := fmt.Sprintf("unrecognised entry type [%s]", fields[leaderFieldID])
			return errors.New(errors.DatabaseError, msg)
		}

		ent, err = deserialise(strings.Split(fields[numLeaderFields], ","))
		if err != nil {
			return err
		}

		db.entries[key] = ent
	}

	return nil
}
