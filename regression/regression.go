package regression

import (
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware"
	"gopher2600/television/digesttv"
	"io"
	"os"
	"strconv"
)

const regressionDBFile = ".gopher2600/regressionDB"

func keyify(cartridgeFile string) (string, error) {
	f, err := os.Open(cartridgeFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	key := sha1.New()
	if _, err := io.Copy(key, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", key.Sum(nil)), nil
}

type regressionEntry struct {
	key           string
	cartridgeFile string
	tvMode        string
	numOFrames    int
	digest        string
}

func (entry regressionEntry) String() string {
	return fmt.Sprintf("%s [%s] frames=%d", entry.cartridgeFile, entry.tvMode, entry.numOFrames)
}

type regressionDB struct {
	dbfile  *os.File
	entries map[string]regressionEntry
}

func (db *regressionDB) endSession() error {
	// write entries to regression database
	csvw := csv.NewWriter(db.dbfile)

	err := db.dbfile.Truncate(0)
	if err != nil {
		return err
	}

	db.dbfile.Seek(0, os.SEEK_SET)

	for _, entry := range db.entries {
		rec := make([]string, 5)
		rec[0] = entry.key
		rec[1] = entry.cartridgeFile
		rec[2] = entry.tvMode
		rec[3] = strconv.Itoa(entry.numOFrames)
		rec[4] = entry.digest

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
	csvr.FieldsPerRecord = 5

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

		numOfFrames, err := strconv.Atoi(rec[3])
		if err != nil {
			return err
		}

		// add entry to database
		entry := regressionEntry{
			key:           rec[0],
			cartridgeFile: rec[1],
			tvMode:        rec[2],
			numOFrames:    numOfFrames,
			digest:        rec[4]}

		db.entries[entry.key] = entry
	}

	return nil
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

func run(cartridgeFile string, tvMode string, numOfFrames int) (string, error) {
	tv, err := digesttv.NewDigestTV(tvMode)
	if err != nil {
		return "", fmt.Errorf("error preparing television: %s", err)
	}

	vcs, err := hardware.NewVCS(tv)
	if err != nil {
		return "", fmt.Errorf("error preparing VCS: %s", err)
	}

	err = vcs.AttachCartridge(cartridgeFile)
	if err != nil {
		return "", err
	}

	err = vcs.RunForFrameCount(numOfFrames)
	if err != nil {
		return "", err
	}

	// output current digest
	return fmt.Sprintf("%s", tv), nil
}

// RegressAddCartridge adds a cartridge to the regression db
func addCartridge(cartridgeFile string, tvMode string, numOfFrames int, allowUpdate bool) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession()

	// run cartdrige and get digest
	digest, err := run(cartridgeFile, tvMode, numOfFrames)
	if err != nil {
		return err
	}

	// add new entry to database
	key, err := keyify(cartridgeFile)
	if err != nil {
		return err
	}
	entry := regressionEntry{
		key:           key,
		cartridgeFile: cartridgeFile,
		tvMode:        tvMode,
		numOFrames:    numOfFrames,
		digest:        digest}

	if allowUpdate == false {
		if existEntry, ok := db.entries[entry.key]; ok {
			if existEntry.cartridgeFile == entry.cartridgeFile {
				return errors.NewGopherError(errors.RegressionEntryExists, entry)
			}

			return errors.NewGopherError(errors.RegressionEntryCollision, entry.cartridgeFile, existEntry.cartridgeFile)
		}
	}

	db.entries[entry.key] = entry

	return nil
}

// RegressDeleteCartridge removes a cartridge from the regression db
func RegressDeleteCartridge(cartridgeFile string) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession()

	key, err := keyify(cartridgeFile)
	if err != nil {
		return err
	}

	if _, ok := db.entries[key]; ok == false {
		return errors.NewGopherError(errors.RegressionEntryDoesNotExist, cartridgeFile)
	}

	delete(db.entries, key)

	return nil
}

// RegressAddCartridge adds a cartridge to the regression db
func RegressAddCartridge(cartridgeFile string, tvMode string, numOfFrames int) error {
	return addCartridge(cartridgeFile, tvMode, numOfFrames, false)
}

// RegressUpdateCartridge updates a entry (or adds it if it doesn't exist)
func RegressUpdateCartridge(cartridgeFile string, tvMode string, numOfFrames int) error {
	return addCartridge(cartridgeFile, tvMode, numOfFrames, true)
}

// RegressRunTests runs the
func RegressRunTests(output io.Writer, failOnError bool) (int, int, error) {
	db, err := startSession()
	if err != nil {
		return -1, -1, err
	}
	defer db.endSession()

	numSucceed := 0
	numFail := 0
	for _, entry := range db.entries {
		digest, err := run(entry.cartridgeFile, entry.tvMode, entry.numOFrames)

		if err != nil || entry.digest != digest {
			if err == nil {
				err = errors.NewGopherError(errors.RegressionEntryFail, entry)
			}

			numFail++
			if failOnError {
				return numSucceed, numFail, err
			}
			if output != nil {
				output.Write([]byte(fmt.Sprintf("fail: %s\n", err)))
			}

		} else {
			numSucceed++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("succeed: %s\n", entry)))
			}
		}
	}

	return numSucceed, numFail, nil
}
