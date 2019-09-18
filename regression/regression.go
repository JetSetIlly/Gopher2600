package regression

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/errors"
	"io"
	"os"
	"sort"
	"strconv"
)

const regressionScripts = ".gopher2600/regressionScripts"
const regressionDBFile = ".gopher2600/regressionDB"

// Regressor represents the generic entry in the regression database
type Regressor interface {
	database.Entry

	// perform the regression test for the regression type. the newRegression
	// flag is for convenience really (or "logical binding", as the structured
	// programmers would have it)
	//
	// message is the string that is to be printed during the regression
	regress(newRegression bool, output io.Writer, message string) (bool, error)
}

// when starting a database session we need to register what entries we will
// find in the database
func initDBSession(db *database.Session) error {
	if err := db.AddEntryType(frameEntryID, deserialiseFrameEntry); err != nil {
		return err
	}

	if err := db.AddEntryType(playbackEntryID, deserialisePlaybackEntry); err != nil {
		return err
	}

	// make sure regression script directory exists
	if err := os.MkdirAll(regressionScripts, 0755); err != nil {
		msg := fmt.Sprintf("regression script directory: %s", err)
		return errors.New(errors.DatabaseError, msg)
	}

	return nil
}

// RegressList displays all entries in the database
func RegressList(output io.Writer) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressList", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := database.StartSession(regressionDBFile, database.ActivityReading, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(false)

	return db.List(output)
}

// RegressDelete removes a cartridge from the regression db
func RegressDelete(output io.Writer, confirmation io.Reader, key string) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressDelete()", "io.Writer should not be nil (use nopWriter)")
	}

	v, err := strconv.Atoi(key)
	if err != nil {
		msg := fmt.Sprintf("invalid key [%s]", key)
		return errors.New(errors.DatabaseError, msg)
	}

	db, err := database.StartSession(regressionDBFile, database.ActivityModifying, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(true)

	reg, err := db.Get(v)
	if err != nil {
		return err
	}

	output.Write([]byte(fmt.Sprintf("%s\ndelete? (y/n): ", reg)))

	confirm := make([]byte, 32)
	_, err = confirmation.Read(confirm)
	if err != nil {
		return err
	}

	if confirm[0] == 'y' || confirm[0] == 'Y' {
		err = db.Delete(reg)
		if err != nil {
			return err
		}
		output.Write([]byte(fmt.Sprintf("deleted test #%s from regression database\n", key)))
	}

	return nil
}

// RegressAdd adds a new regression handler to the database
func RegressAdd(output io.Writer, reg Regressor) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressAdd()", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := database.StartSession(regressionDBFile, database.ActivityCreating, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(true)

	msg := fmt.Sprintf("adding: %s", reg)
	ok, err := reg.regress(true, output, msg)
	if !ok || err != nil {
		return err
	}

	output.Write([]byte(ansi.ClearLine))
	output.Write([]byte(fmt.Sprintf("\radded: %s\n", reg)))

	return db.Add(reg)
}

// RegressRunTests runs all the tests in the regression database
// o filterKeys list specified which entries to test. an empty keys list means that
//	every entry should be tested
func RegressRunTests(output io.Writer, verbose bool, failOnError bool, filterKeys []string) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressRunEntries()", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := database.StartSession(regressionDBFile, database.ActivityReading, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(false)

	// make sure any supplied keys list is in order
	keysV := make([]int, 0, len(filterKeys))
	for k := range filterKeys {
		v, err := strconv.Atoi(filterKeys[k])
		if err != nil {
			msg := fmt.Sprintf("invalid key [%s]", filterKeys[k])
			return errors.New(errors.DatabaseError, msg)
		}
		keysV = append(keysV, v)
	}
	sort.Ints(keysV)
	filterIdx := 0

	numSucceed := 0
	numFail := 0
	numError := 0
	numSkipped := 0

	defer func() {
		output.Write([]byte(fmt.Sprintf("regression tests: %d succeed, %d fail, %d skipped", numSucceed, numFail, numSkipped)))

		if numError > 0 {
			output.Write([]byte(" [with errors]"))
		}
		output.Write([]byte("\n"))
	}()

	onSelect := func(ent database.Entry) (bool, error) {
		key := ent.GetKey()

		// if a list of keys has been supplied then check key in the database
		// against that list (both lists are sorted)
		if len(keysV) > 0 {
			// if we've come to the end of the list of filter keys then update
			// the number of skipped entries and return false to indicate that
			// the Select() function should not continue
			if filterIdx >= len(keysV) {
				numSkipped += db.NumEntries() - key
				return false, nil
			}

			// if entry key is not in list of keys then update number of
			// skipped entries and return true to indicate that the Select()
			// function should countinue
			if keysV[filterIdx] != key {
				numSkipped++
				return true, nil
			}

			// entry key is in list: because we're receiving database entries
			// in order and because the list of filter keys is also sorted, we
			// can bump the filterIdx to the next entry
			filterIdx++
		}

		// datbase entry should also satisfy Regressor interface
		reg, ok := ent.(Regressor)
		if !ok {
			return false, errors.New(errors.PanicError, "database entry does not satisfy Regressor interface")
		}

		// run regress() function with message. message does not have a
		// trailing newline
		msg := fmt.Sprintf("running: %s", reg)
		ok, err := reg.regress(false, output, msg)

		// once regress() has completed we clear the line ready for the
		// completion message
		output.Write([]byte(ansi.ClearLine))

		// print completion message depending on result of regress()
		if err != nil {
			numError++
			output.Write([]byte(fmt.Sprintf("\r ERROR: %s\n", reg)))

			// output any error message on following line
			if verbose {
				output.Write([]byte(fmt.Sprintf("%s\n", err)))
			}

			if failOnError {
				return false, nil
			}
		} else if !ok {
			numFail++
			output.Write([]byte(fmt.Sprintf("\rfailure: %s\n", reg)))

		} else {
			numSucceed++
			output.Write([]byte(fmt.Sprintf("\rsucceed: %s\n", reg)))
		}

		return true, nil
	}

	db.Select("", onSelect)

	return nil
}
