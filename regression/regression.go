package regression

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"strconv"
)

// RegressList displays all entries in the database
func RegressList(output io.Writer) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(false)

	db.listRecords(output)

	return nil
}

// RegressDelete removes a cartridge from the regression db
func RegressDelete(key string) error {
	v, err := strconv.Atoi(key)
	if err != nil {
		msg := fmt.Sprintf("invalid key [%s]", key)
		return errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	// TODO: display record and ask for confirmatio

	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(true)

	return db.delRecord(v)
}

// RegressAdd adds a cartridge or run-recording to the regression db
func RegressAdd(rec record) error {
	_, err := rec.regress(true)
	if err != nil {
		return err
	}

	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(true)

	return db.addRecord(rec)
}

// RegressRunTests runs all the tests in the regression database
func RegressRunTests(output io.Writer) (int, int, error) {
	db, err := startSession()
	if err != nil {
		return -1, -1, err
	}
	defer db.endSession(false)

	numSucceed := 0
	numFail := 0
	for _, key := range db.keys {
		rec := db.records[key]

		ok, err := rec.regress(false)
		if err != nil {
			return numSucceed, numFail, errors.NewFormattedError(errors.RegressionFail, rec.String())
		}

		if !ok {
			numFail++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("fail: %s\n", rec)))
			}

		} else {
			numSucceed++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("succeed: %s\n", rec)))
			}
		}
	}

	return numSucceed, numFail, nil
}
