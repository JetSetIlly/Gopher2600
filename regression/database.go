package regression

import (
	"encoding/csv"
	"fmt"
	"gopher2600/errors"
	"io"
	"os"
	"strconv"
)

const regressionDBFile = ".gopher2600/regressionDB"

type regressionEntry struct {
	cartridgePath string
	tvMode        string
	numOFrames    int
	screenDigest  string
}

const numFields = 4

func (entry regressionEntry) String() string {
	return fmt.Sprintf("%s [%s] frames=%d", entry.cartridgePath, entry.tvMode, entry.numOFrames)
}

type regressionDB struct {
	dbfile  *os.File
	entries map[string]regressionEntry
}

func startSession() (*regressionDB, error) {
	var err error

	db := &regressionDB{}

	db.dbfile, err = os.OpenFile(regressionDBFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	err = db.readEntries()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *regressionDB) endSession(commitChanges bool) error {
	// write entries to regression database
	if commitChanges {
		csvw := csv.NewWriter(db.dbfile)

		err := db.dbfile.Truncate(0)
		if err != nil {
			return err
		}

		db.dbfile.Seek(0, os.SEEK_SET)

		for _, entry := range db.entries {
			rec := make([]string, numFields)
			rec[0] = entry.cartridgePath
			rec[1] = entry.tvMode
			rec[2] = strconv.Itoa(entry.numOFrames)
			rec[3] = entry.screenDigest

			err := csvw.Write(rec)
			if err != nil {
				return err
			}
		}

		// make sure everything's been written
		csvw.Flush()
		err = csvw.Error()
		if err != nil {
			return err
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

func (db *regressionDB) readEntries() error {
	// readEntries clobbers the contents of db.entries
	db.entries = make(map[string]regressionEntry, len(db.entries))

	// treat the file as a CSV file
	csvr := csv.NewReader(db.dbfile)
	csvr.Comment = rune('#')
	csvr.TrimLeadingSpace = true
	csvr.ReuseRecord = true
	csvr.FieldsPerRecord = numFields

	db.dbfile.Seek(0, os.SEEK_SET)

	for {
		// loop through file until EOF is reached
		rec, err := csvr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		numOfFrames, err := strconv.Atoi(rec[2])
		if err != nil {
			return err
		}

		// add entry to database
		entry := regressionEntry{
			cartridgePath: rec[0],
			tvMode:        rec[1],
			numOFrames:    numOfFrames,
			screenDigest:  rec[3]}

		db.entries[entry.cartridgePath] = entry
	}

	return nil
}

// RegressAddCartridge adds a cartridge to the regression db
func addCartridge(cartridgeFile string, tvMode string, numOfFrames int, allowUpdate bool) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(true)

	// run cartdrige and get digest
	digest, err := run(cartridgeFile, tvMode, numOfFrames)
	if err != nil {
		return err
	}

	entry := regressionEntry{
		cartridgePath: cartridgeFile,
		tvMode:        tvMode,
		numOFrames:    numOfFrames,
		screenDigest:  digest}

	if allowUpdate == false {
		if existEntry, ok := db.entries[entry.cartridgePath]; ok {
			if existEntry.cartridgePath == entry.cartridgePath {
				return errors.NewFormattedError(errors.RegressionEntryExists, entry)
			}

			return errors.NewFormattedError(errors.RegressionEntryCollision, entry.cartridgePath, existEntry.cartridgePath)
		}
	}

	db.entries[entry.cartridgePath] = entry

	return nil
}
