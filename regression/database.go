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
const regressionScripts = ".gopher2600/regressionScripts"
const fieldSep = ","
const regressionSep = "\n"

// arbitrary number of regressions
const maxRegressions = 1000

type regressionDB struct {
	dbfile      *os.File
	regressions map[int]Handler

	// sorted list of keys. used for:
	// - displaying regressions in correct order in list()
	// - saving in correct order in endSession()
	keys []int
}

const (
	leaderFieldKey int = iota
	leaderFieldID
	numLeaderFields
)

// csvLeader returns the first two fields required for every regression type
func csvLeader(reg Handler) string {
	return fmt.Sprintf("%03d%s%s", reg.getKey(), fieldSep, reg.getID())
}

func startSession() (*regressionDB, error) {
	var err error

	db := &regressionDB{}

	db.dbfile, err = os.OpenFile(regressionDBFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	err = db.readDBFile()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *regressionDB) endSession(commitChanges bool) error {
	// write regressions to database
	if commitChanges {
		err := db.dbfile.Truncate(0)
		if err != nil {
			return err
		}

		db.dbfile.Seek(0, os.SEEK_SET)

		for _, key := range db.keys {
			db.dbfile.WriteString(db.regressions[key].getCSV())
			db.dbfile.WriteString(regressionSep)
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

func (db *regressionDB) readDBFile() error {
	// clobbers the contents of db.regressions
	db.regressions = make(map[int]Handler, len(db.regressions))

	// make sure we're at the beginning of the file
	db.dbfile.Seek(0, os.SEEK_SET)

	buffer, err := ioutil.ReadAll(db.dbfile)
	if err != nil {
		return errors.NewFormattedError(errors.RegressionDBError, err)
	}

	// split regressions
	lines := strings.Split(string(buffer), regressionSep)

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

		if _, ok := db.regressions[key]; ok {
			msg := fmt.Sprintf("duplicate key [%v] at line %d", key, i+1)
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		var reg Handler

		switch fields[leaderFieldID] {
		case "frame":
			reg, err = newFrameRegression(key, fields[numLeaderFields])
			if err != nil {
				return err
			}
		case "playback":
			reg, err = newPlaybackRegression(key, fields[numLeaderFields])
			if err != nil {
				return err
			}
		default:
			msg := fmt.Sprintf("unrecognised regression type [%s]", fields[leaderFieldID])
			return errors.NewFormattedError(errors.RegressionDBError, msg)
		}

		db.regressions[key] = reg

		// add key to list
		db.keys = append(db.keys, key)
	}

	// sort key list
	sort.Ints(db.keys)

	return nil
}

func (db regressionDB) list(output io.Writer) {
	for k := range db.keys {
		output.Write([]byte(fmt.Sprintf("%03d ", db.keys[k])))
		output.Write([]byte(db.regressions[db.keys[k]].String()))
		output.Write([]byte("\n"))
	}
	if len(db.keys) == 0 {
		output.Write([]byte("regression DB is empty\n"))
	} else {
		output.Write([]byte(fmt.Sprintf("Total: %d\n", len(db.keys))))
	}
}

// add adds a cartridge to the regression db
func (db *regressionDB) add(reg Handler) error {
	var key int

	// find spare key
	for key = 0; key < maxRegressions; key++ {
		if _, ok := db.regressions[key]; !ok {
			break
		}
	}

	if key == maxRegressions {
		msg := fmt.Sprintf("%d regression maximum exceeded", maxRegressions)
		return errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	reg.setKey(key)
	db.regressions[key] = reg

	// add key to list and resort
	db.keys = append(db.keys, key)
	sort.Ints(db.keys)

	return nil
}

func (db *regressionDB) del(reg Handler) error {
	reg.cleanUp()

	delete(db.regressions, reg.getKey())

	// find key in list and delete
	for i := 0; i < len(db.keys); i++ {
		if db.keys[i] == reg.getKey() {
			db.keys = append(db.keys[:i], db.keys[i+1:]...)
			break // for loop
		}
	}

	return nil
}

func (db regressionDB) get(key int) (Handler, error) {
	reg, ok := db.regressions[key]
	if !ok {
		msg := fmt.Sprintf("key not found [%d]", key)
		return nil, errors.NewFormattedError(errors.RegressionDBError, msg)
	}
	return reg, nil
}
