package regression

import (
	"fmt"
	"gopher2600/errors"
	"io"
	"sort"
	"strconv"
)

// Handler represents the generic entry in the regression database
type Handler interface {
	// getID returns the string that is used to identify the regression type in
	// the database
	getID() string

	// setKey sets the key value for the regression
	setKey(int)

	// getKey returns the key assigned to the regression
	getKey() int

	// getCSV returns the comma separated string representing the regression.
	// without regression separator. the first two fields should be the result of
	// csvLeader()
	getCSV() string

	// Run performs the regression test for the regression type
	regress(newRegression bool) (bool, error)

	// action perfomed when regression entry is removed from database. for
	// example, removing additional files from disk
	cleanUp()

	// String implements the Stringer interface
	String() string
}

// RegressList displays all entries in the database
func RegressList(output io.Writer) error {
	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(false)

	db.list(output)

	return nil
}

// RegressDelete removes a cartridge from the regression db
func RegressDelete(output io.Writer, confirmation io.Reader, key string) (bool, error) {
	v, err := strconv.Atoi(key)
	if err != nil {
		msg := fmt.Sprintf("invalid key [%s]", key)
		return false, errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	db, err := startSession()
	if err != nil {
		return false, err
	}
	defer db.endSession(true)

	reg, err := db.get(v)
	if err != nil {
		return false, err
	}

	confirm := make([]byte, 1)
	output.Write([]byte(fmt.Sprintf("delete? (y/n): %s\n", reg)))

	_, err = confirmation.Read(confirm)
	if err != nil {
		return false, err
	}

	if confirm[0] == 'y' || confirm[0] == 'Y' {
		err = db.del(reg)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

// RegressAdd adds a new regression handler to the database
func RegressAdd(output io.Writer, reg Handler) error {
	output.Write([]byte(fmt.Sprintf("adding: %s\r", reg)))
	ok, err := reg.regress(true)
	if !ok || err != nil {
		return err
	}
	output.Write([]byte(fmt.Sprintf("added: %s\r", reg)))

	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(true)

	return db.add(reg)
}

// RegressRunTests runs all the tests in the regression database
// keys list specified which entries to test. an empty keys list means that
// every entry should be tested
func RegressRunTests(output io.Writer, keys []string) (int, int, int, error) {
	db, err := startSession()
	if err != nil {
		return -1, -1, -1, err
	}
	defer db.endSession(false)

	// make sure any supplied keys list is in order
	keysV := make([]int, 0, len(keys))
	for k := range keys {
		v, err := strconv.Atoi(keys[k])
		if err != nil {
			msg := fmt.Sprintf("invalid key [%s]", keys[k])
			return -1, -1, -1, errors.NewFormattedError(errors.RegressionDBError, msg)
		}
		keysV = append(keysV, v)
	}
	sort.Ints(keysV)
	k := 0

	numSucceed := 0
	numFail := 0
	numSkipped := 0
	for key := 0; key < len(db.keys); key++ {

		// if a list of keys has been supplied then check key in the database
		// against that list (both lists are sorted
		if len(keysV) > 0 {
			if k >= len(keysV) {
				numSkipped += len(db.keys) - key
				break // for loop
			}
			if keysV[k] != key {
				numSkipped++
				continue // for loop
			}
			k++
		}

		reg := db.regressions[db.keys[key]]

		output.Write([]byte(fmt.Sprintf("running: %s\r", reg)))
		ok, err := reg.regress(false)

		if !ok || err != nil {
			numFail++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("failure: %s\n", reg)))

				// output any error message on following line
				if err != nil {
					output.Write([]byte(fmt.Sprintf("%s\n", err)))
				}
			}

		} else {
			numSucceed++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("succeed: %s\n", reg)))
			}
		}
	}

	return numSucceed, numFail, numSkipped, nil
}
