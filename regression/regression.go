package regression

import (
	"fmt"
	"gopher2600/debugger/colorterm/ansi"
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

	// set the key value for the regression
	setKey(int)

	// return the key assigned to the regression
	getKey() int

	// return the comma separated string representing the regression.  without
	// regression separator. the first two fields should be the result of
	// csvLeader()
	generateCSV() string

	// perform the regression test for the regression type. the newRegression
	// flag is for convenience really (or "logical binding", as the structured
	// programmers would have it)
	//
	// message is the string that is to be printed during the regression
	regress(newRegression bool, output io.Writer, message string) (bool, error)

	// action perfomed when regression entry is removed from database
	cleanUp()

	// String implements the Stringer interface
	String() string
}

// RegressList displays all entries in the database
func RegressList(output io.Writer) error {
	if output == nil {
		return errors.NewFormattedError(errors.FatalError, "RegressList", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(false)

	db.list(output)

	return nil
}

// RegressDelete removes a cartridge from the regression db
func RegressDelete(output io.Writer, confirmation io.Reader, key string) error {
	if output == nil {
		return errors.NewFormattedError(errors.FatalError, "RegressDelete()", "io.Writer should not be nil (use nopWriter)")
	}

	v, err := strconv.Atoi(key)
	if err != nil {
		msg := fmt.Sprintf("invalid key [%s]", key)
		return errors.NewFormattedError(errors.RegressionDBError, msg)
	}

	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(true)

	reg, err := db.get(v)
	if err != nil {
		return err
	}

	output.Write([]byte(fmt.Sprintf("%s\ndelete? (y/n): ", reg)))

	confirm := make([]byte, 1024)
	_, err = confirmation.Read(confirm)
	if err != nil {
		return err
	}

	if confirm[0] == 'y' || confirm[0] == 'Y' {
		err = db.del(reg)
		if err != nil {
			return err
		}
		output.Write([]byte(fmt.Sprintf("deleted test #%s from regression database\n", key)))
	}

	return nil
}

// RegressAdd adds a new regression handler to the database
func RegressAdd(output io.Writer, reg Handler) error {
	if output == nil {
		return errors.NewFormattedError(errors.FatalError, "RegressAdd()", "io.Writer should not be nil (use nopWriter)")
	}

	msg := fmt.Sprintf("adding: %s", reg)
	ok, err := reg.regress(true, output, msg)
	if !ok || err != nil {
		return err
	}

	output.Write([]byte(ansi.ClearLine))
	output.Write([]byte(fmt.Sprintf("\radded: %s\n", reg)))

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
func RegressRunTests(output io.Writer, keys []string) error {
	if output == nil {
		return errors.NewFormattedError(errors.FatalError, "RegressRunEntries()", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := startSession()
	if err != nil {
		return err
	}
	defer db.endSession(false)

	// make sure any supplied keys list is in order
	keysV := make([]int, 0, len(keys))
	for k := range keys {
		v, err := strconv.Atoi(keys[k])
		if err != nil {
			msg := fmt.Sprintf("invalid key [%s]", keys[k])
			return errors.NewFormattedError(errors.RegressionDBError, msg)
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

		msg := fmt.Sprintf("running: %s", reg)
		ok, err := reg.regress(false, output, msg)
		output.Write([]byte(ansi.ClearLine))

		if !ok || err != nil {
			numFail++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("\rfailure: %s\n", reg)))

				// output any error message on following line
				if err != nil {
					output.Write([]byte(fmt.Sprintf("\t%s\n", err)))
				}
			}

		} else {
			numSucceed++
			if output != nil {
				output.Write([]byte(fmt.Sprintf("\rsucceed: %s\n", reg)))
			}
		}
	}

	output.Write([]byte(fmt.Sprintf("regression tests: %d succeed, %d fail, %d skipped\n", numSucceed, numFail, numSkipped)))

	return nil
}
