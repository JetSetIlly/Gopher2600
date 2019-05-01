package regression

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

const regressionDBFile = ".gopher2600/regressionDB"
const fieldSep = ","
const recordSep = "\n"

// arbitrary number of records
const maxRecords = 1000

type regressionDB struct {
	dbfile  *os.File
	records map[int]record

	// sorted list of keys. used for:
	// - displaying records in correct order in listRecords()
	// - saving in correct order in endSession()
	keys []int
}

type record interface {
	// getID returns the string that is used to identify the record type in the
	// database
	getID() string

	// String implements the Stringer interface
	String() string

	// setKey sets the key value for the record
	setKey(int)

	// getKey returns the key assigned to the record
	getKey() int

	// getCSV returns the comma separated string representing the record.
	// without record separator. the first two fields should be the result of
	// csvRecordLeader()
	getCSV() string

	// Run performs the regression test for the record type
	regress(newRecord bool) (bool, error)
}

const (
	leaderFieldKey int = iota
	leaderFieldID
	numLeaderFields
)

// csvRecordLeader returns the first two fields required for every record type
func csvLeader(rec record) string {
	return fmt.Sprintf("%03d%s%s", rec.getKey(), fieldSep, rec.getID())
}

func startSession() (*regressionDB, error) {
	var err error

	db := &regressionDB{}

	db.dbfile, err = os.OpenFile(regressionDBFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	err = db.readRecords()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *regressionDB) endSession(commitChanges bool) error {
	// write records to regression database
	if commitChanges {
		err := db.dbfile.Truncate(0)
		if err != nil {
			return err
		}

		db.dbfile.Seek(0, os.SEEK_SET)

		for _, key := range db.keys {
			db.dbfile.WriteString(db.records[key].getCSV())
			db.dbfile.WriteString(recordSep)
		}
	}

	// end session by closing file
	if db.dbfile != nil {
		if err := db.dbfile.Close(); err != nil {
			return err
		}
		db.dbfile = nil
	}

	return nil
}

func (db *regressionDB) readRecords() error {
	// readrecords clobbers the contents of db.entrie
	db.records = make(map[int]record, len(db.records))

	// make sure we're at the beginning of the file
	db.dbfile.Seek(0, os.SEEK_SET)

	buffer, err := ioutil.ReadAll(db.dbfile)
	if err != nil {
		return errors.NewFormattedError(errors.RegressionDBError, err)
	}

	// split records
	lines := strings.Split(string(buffer), recordSep)

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

		if _, ok := db.records[key]; ok {
			msg := fmt.Sprintf("duplicate key [%v] at line %d", key, i+1)
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		var rec record

		switch fields[leaderFieldID] {
		case "frame":
			rec, err = newFrameRecord(key, fields[numLeaderFields])
			if err != nil {
				return err
			}
		default:
			msg := fmt.Sprintf("unrecognised record type [%s]", fields[leaderFieldID])
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		db.records[key] = rec

		// add key to list
		db.keys = append(db.keys, key)
	}

	// sort key list
	sort.Ints(db.keys)

	return nil
}

func (db regressionDB) listRecords(output io.Writer) {
	for k := range db.keys {
		output.Write([]byte(fmt.Sprintf("%03d [%s] ", db.keys[k], db.records[db.keys[k]].getID())))
		output.Write([]byte(db.records[db.keys[k]].String()))
		output.Write([]byte("\n"))
	}
}

// addRecord adds a cartridge to the regression db
func (db *regressionDB) addRecord(rec record) error {
	var key int

	// find spare key
	for key = 0; key < maxRecords; key++ {
		if _, ok := db.records[key]; !ok {
			break
		}
	}

	if key == maxRecords {
		msg := fmt.Sprintf("%d record maximum exceeded", maxRecords)
		return errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	rec.setKey(key)
	db.records[key] = rec

	// add key to list and resort
	db.keys = append(db.keys, key)
	sort.Ints(db.keys)

	return nil
}

func (db *regressionDB) delRecord(key int) error {
	if _, ok := db.records[key]; ok == false {
		msg := fmt.Sprintf("key not found [%d]", key)
		return errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	delete(db.records, key)

	// find key in list and delete
	for i := 0; i < len(db.keys); i++ {
		if db.keys[i] == key {
			db.keys = append(db.keys[:i], db.keys[i+1:]...)
			break // for loop
		}
	}

	return nil
}
