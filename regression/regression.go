package regression

import (
	"fmt"
	"gopher2600/database"
	"gopher2600/debugger/colorterm/ansi"
	"gopher2600/errors"
	"gopher2600/paths"
	"io"
	"os"
	"sort"
	"strconv"
)

// the location of the regressionDB file and the location of any regression
// scripts. these should be wrapped by paths.ResourcePath()
const regressionDBFile = "regressionDB"
const regressionScripts = "regressionScripts"

// Regressor is the generic entry type in the regressionDB
//
// * exported because we use it directly from the main package
type Regressor interface {
	database.Entry

	// perform the regression test for the regression type. the newRegression
	// flag is for convenience really (or "logical binding", as the structured
	// programmers would have it)
	//
	// message is the string that is to be printed during the regression
	//
	// returns: success boolean; any failure message (not always appropriate;
	// and error state
	regress(newRegression bool, output io.Writer, message string) (bool, string, error)
}

// when starting a database session we need to register what entries we will
// find in the database
func initDBSession(db *database.Session) error {
	if err := db.RegisterEntryType(digestEntryID, deserialiseDigestEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(playbackEntryID, deserialisePlaybackEntry); err != nil {
		return err
	}

	// make sure regression script directory exists
	if err := os.MkdirAll(paths.ResourcePath(regressionScripts), 0755); err != nil {
		msg := fmt.Sprintf("regression script directory: %s", err)
		return errors.New(errors.RegressionError, msg)
	}

	return nil
}

// RegressList displays all entries in the database
func RegressList(output io.Writer) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressList()", "io.Writer should not be nil (use a nopWriter)")
	}

	db, err := database.StartSession(paths.ResourcePath(regressionDBFile), database.ActivityReading, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(false)

	return db.List(output)
}

// RegressAdd adds a new regression handler to the database
func RegressAdd(output io.Writer, reg Regressor) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressAdd()", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := database.StartSession(paths.ResourcePath(regressionDBFile), database.ActivityCreating, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(true)

	msg := fmt.Sprintf("adding: %s", reg)
	_, _, err = reg.regress(true, output, msg)
	if err != nil {
		return err
	}

	output.Write([]byte(ansi.ClearLine))
	output.Write([]byte(fmt.Sprintf("\radded: %s\n", reg)))

	return db.Add(reg)
}

// RegressDelete removes a cartridge from the regression db
func RegressDelete(output io.Writer, confirmation io.Reader, key string) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressDelete()", "io.Writer should not be nil (use nopWriter)")
	}

	v, err := strconv.Atoi(key)
	if err != nil {
		msg := fmt.Sprintf("invalid key [%s]", key)
		return errors.New(errors.RegressionError, msg)
	}

	db, err := database.StartSession(paths.ResourcePath(regressionDBFile), database.ActivityModifying, initDBSession)
	if err != nil {
		return err
	}
	defer db.EndSession(true)

	ent, err := db.SelectKeys(nil, v)
	if err != nil {
		if !errors.Is(err, errors.DatabaseSelectEmpty) {
			return err
		}

		// select returned no entries; create DatabaseKeyError and wrap it in a
		// RegressionError
		return errors.New(errors.RegressionError, errors.New(errors.DatabaseKeyError, v))
	}

	output.Write([]byte(fmt.Sprintf("%s\ndelete? (y/n): ", ent)))

	confirm := make([]byte, 32)
	_, err = confirmation.Read(confirm)
	if err != nil {
		return err
	}

	if confirm[0] == 'y' || confirm[0] == 'Y' {
		err = db.Delete(v)
		if err != nil {
			fmt.Println(1)
			return err
		}
		output.Write([]byte(fmt.Sprintf("deleted test #%s from regression database\n", key)))
	}

	return nil
}

// RegressRunTests runs all the tests in the regression database
// o filterKeys list specified which entries to test. an empty keys list means that
//	every entry should be tested
func RegressRunTests(output io.Writer, verbose bool, failOnError bool, filterKeys []string) error {
	if output == nil {
		return errors.New(errors.PanicError, "RegressRunEntries()", "io.Writer should not be nil (use nopWriter)")
	}

	db, err := database.StartSession(paths.ResourcePath(regressionDBFile), database.ActivityReading, initDBSession)
	if err != nil {
		return errors.New(errors.RegressionError, err)
	}
	defer db.EndSession(false)

	// make sure any supplied keys list is in order
	keysV := make([]int, 0, len(filterKeys))
	for k := range filterKeys {
		v, err := strconv.Atoi(filterKeys[k])
		if err != nil {
			msg := fmt.Sprintf("invalid key [%s]", filterKeys[k])
			return errors.New(errors.RegressionError, msg)
		}
		keysV = append(keysV, v)
	}
	sort.Ints(keysV)

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
		// datbase entry should also satisfy Regressor interface
		reg, ok := ent.(Regressor)
		if !ok {
			return false, errors.New(errors.PanicError, "RegressRunTests()", "database entry does not satisfy Regressor interface")
		}

		// run regress() function with message. message does not have a
		// trailing newline
		msg := fmt.Sprintf("running: %s", reg)
		ok, failm, err := reg.regress(false, output, msg)

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
			if verbose && failm != "" {
				output.Write([]byte(fmt.Sprintf("  ^^ %s\n", failm)))
			}

		} else {
			numSucceed++
			output.Write([]byte(fmt.Sprintf("\rsucceed: %s\n", reg)))
		}

		return true, nil
	}

	db.SelectKeys(onSelect, keysV...)

	return nil
}
