// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package regression

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/database"
	"github.com/jetsetilly/gopher2600/resources"
)

// ansi code for clear line.
const ansiClearLine = "\033[2K"

// the location of the regressionDB file and the location of any regression
// scripts. these should be wrapped by resources.JoinPath().
const regressionPath = "regression"
const regressionDBFile = "db"
const regressionScripts = "scripts"
const fails = "fails"

// Regressor is the generic entry type in the regressionDB.
type Regressor interface {
	database.Entry

	// String should return information about the entry in a human readable
	// format. by contrast, machine readable representation is returned by the
	// Serialise function
	String() string

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
// find in the database.
func initDBSession(db *database.Session) error {
	if err := db.RegisterEntryType(videoEntryType, deserialiseVideoEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(playbackEntryType, deserialisePlaybackEntry); err != nil {
		return err
	}

	if err := db.RegisterEntryType(logEntryType, deserialiseLogEntry); err != nil {
		return err
	}

	return nil
}

// RegressList displays all entries in the database.
func RegressList(output io.Writer, keys []string) error {
	if output == nil {
		return fmt.Errorf("regression: io.Writer should not be nil")
	}

	dbPth, err := resources.JoinPath(regressionPath, regressionDBFile)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityReading, initDBSession)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}
	defer db.EndSession(false)

	keys, err = addFailsToKeys(keys)
	if err != nil {
		if errors.Is(err, noPreviousFails) {
			// print message about there being no previous fails after everything else
			defer output.Write([]byte(fmt.Sprintf("no previous fails to list\n")))

			// if other keys have been specified allow them to be run. if the
			// list of keys is empty then that is treated as being all keys,
			// which in the context of "FAILS" being a key is probably not what
			// the user wants
			if len(keys) == 0 {
				return nil
			}
		} else {
			return fmt.Errorf("regression: %w", err)
		}
	}

	onSelect := func(e database.Entry, key int) error {
		output.Write([]byte(fmt.Sprintf("%03d %v\n", key, e)))
		return nil
	}

	keysInt, err := convertKeys(keys)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	_, err = db.SelectKeys(onSelect, keysInt...)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	return nil
}

// RegressAdd adds a new regression handler to the database.
func RegressAdd(output io.Writer, reg Regressor) error {
	if output == nil {
		return fmt.Errorf("regression: io.Writer should not be nil")
	}

	dbPth, err := resources.JoinPath(regressionPath, regressionDBFile)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityCreating, initDBSession)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}
	defer db.EndSession(true)

	msg := fmt.Sprintf("adding: %s", reg)
	_, _, err = reg.regress(true, output, msg)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	output.Write([]byte(ansiClearLine))
	output.Write([]byte(fmt.Sprintf("\radded: %s\n", reg)))

	return db.Add(reg)
}

// RegressRedux removes and adds an entry using the same parameters.
func RegressRedux(output io.Writer, confirmation io.Reader, dryRun bool, keys []string) error {
	if output == nil {
		return fmt.Errorf("regression: io.Writer should not be nil")
	}

	if confirmation == nil {
		return fmt.Errorf("regression: io.Reader should not be nil")
	}

	if !dryRun {
		output.Write([]byte("redux is a dangerous operation. it will rerun all compatible regression entries.\n"))

		output.Write([]byte("redux? (y/n): "))
		if !confirm(confirmation) {
			return nil
		}

		output.Write([]byte("sure? (y/n): "))
		if !confirm(confirmation) {
			return nil
		}
	}

	dbPth, err := resources.JoinPath(regressionPath, regressionDBFile)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityCreating, initDBSession)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}
	defer db.EndSession(true)

	keys, err = addFailsToKeys(keys)
	if err != nil {
		if errors.Is(err, noPreviousFails) {
			// print message about there being no previous fails after everything else
			defer output.Write([]byte(fmt.Sprintf("no previous fails to redux\n")))

			// if other keys have been specified allow them to be run. if the
			// list of keys is empty then that is treated as being all keys,
			// which in the context of "FAILS" being a key is probably not what
			// the user wants
			if len(keys) == 0 {
				return nil
			}
		} else {
			return fmt.Errorf("regression: %w", err)
		}
	}

	// selectKeys() calls this onSelect function for every key entry
	onSelect := func(e database.Entry, key int) error {
		switch reg := e.(type) {
		case *VideoRegression:
			err = redux(db, output, key, reg, dryRun)
			if err != nil {
				return fmt.Errorf("regression: %w", err)
			}

		case *LogRegression:
			err = redux(db, output, key, reg, dryRun)
			if err != nil {
				return fmt.Errorf("regression: %w", err)
			}

		default:
			output.Write([]byte(fmt.Sprintf("skipped: %s\n", reg)))
		}

		return nil
	}

	keysInt, err := convertKeys(keys)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	_, err = db.SelectKeys(onSelect, keysInt...)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	return err
}

func redux(db *database.Session, output io.Writer, key int, reg Regressor, dryRun bool) error {
	var err error

	if !dryRun {
		err = db.Delete(key)
		if err != nil {
			return err
		}
	}

	msg := fmt.Sprintf("reduxing: %s", reg)

	_, _, err = reg.regress(true, output, msg)
	if err != nil {
		return err
	}

	output.Write([]byte(ansiClearLine))
	output.Write([]byte(fmt.Sprintf("\rreduxed: %s\n", reg)))

	if !dryRun {
		err = db.Add(reg)
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanupScript removes orphaned script files from disk. An orphaned file is
// one that exists on disk but has no reference in the regression database
// file.
func RegressCleanup(output io.Writer, confirmation io.Reader) error {
	if output == nil {
		return fmt.Errorf("regression: io.Writer should not be nil")
	}

	if confirmation == nil {
		return fmt.Errorf("regression: io.Reader should not be nil")
	}

	output.Write([]byte("cleanup is a dangerous operation. it will delete all orphaned script files.\n"))

	output.Write([]byte("cleanup? (y/n): "))
	if !confirm(confirmation) {
		return nil
	}

	dbPth, err := resources.JoinPath(regressionPath, regressionDBFile)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityReading, initDBSession)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}
	defer db.EndSession(false)

	// gather list of all files referenced
	filesReferenced := make([]string, 0)

	err = db.ForEach(func(key int, e database.Entry) error {
		switch reg := e.(type) {
		case *VideoRegression:
			if len(strings.TrimSpace(reg.stateFile)) > 0 {
				filesReferenced = append(filesReferenced, reg.stateFile)
			}

		case *PlaybackRegression:
			filesReferenced = append(filesReferenced, reg.Script)

		case *LogRegression:
			// no support required

		default:
			return fmt.Errorf("not supported (%s)", reg.EntryType())
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	// gather list of files on disk in path
	scriptPth, err := resources.JoinPath(regressionPath, regressionScripts)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	var filesOnDisk []os.DirEntry
	filesOnDisk, err = os.ReadDir(scriptPth)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	// prepare statistics
	numDeleted := 0
	numRemain := len(filesReferenced)
	numErrors := 0

	defer func() {
		output.Write([]byte(fmt.Sprintf("regression cleanup: %d deleted, %d remain, %d errors\n", numDeleted, numRemain, numErrors)))
	}()

	// delete any files on disk that are not referenced
	for _, e := range filesOnDisk {
		found := false

		n, err := resources.JoinPath(regressionPath, regressionScripts, e.Name())
		if err != nil {
			return fmt.Errorf("regression: %w", err)
		}

		for _, f := range filesReferenced {
			if f == n {
				found = true
				break // for range filesRefrenced
			}
		}

		if !found {
			output.Write([]byte(fmt.Sprintf("delete %s? (y/n): ", n)))
			if confirm(confirmation) {
				output.Write([]byte(ansiClearLine))
				err := os.Remove(n)
				if err != nil {
					output.Write([]byte(fmt.Sprintf("\rerror deleting: %s (%s)\n", n, err)))
					numErrors++
				} else {
					output.Write([]byte(fmt.Sprintf("\rdeleted: %s\n", n)))
					numDeleted++
					numRemain--
				}
			} else {
				output.Write([]byte(ansiClearLine))
				output.Write([]byte(fmt.Sprintf("\rnot deleting: %s\n", n)))
			}
		} else {
			output.Write([]byte(fmt.Sprintf("\ris referenced: %s\n", n)))
		}
	}

	return nil
}

// RegressDelete removes a cartridge from the regression db.
func RegressDelete(output io.Writer, confirmation io.Reader, key string) error {
	if output == nil {
		return fmt.Errorf("regression: io.Writer should not be nil")
	}

	if confirmation == nil {
		return fmt.Errorf("regression: io.Reader should not be nil")
	}

	v, err := strconv.Atoi(key)
	if err != nil {
		return fmt.Errorf("regression: invalid key [%s]", key)
	}

	dbPth, err := resources.JoinPath(regressionPath, regressionDBFile)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityModifying, initDBSession)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}
	defer db.EndSession(true)

	ent, err := db.SelectKeys(nil, v)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	output.Write([]byte(fmt.Sprintf("%s\ndelete? (y/n): ", ent)))
	if confirm(confirmation) {
		err = db.Delete(v)
		if err != nil {
			return fmt.Errorf("regression: %w", err)
		}
		output.Write([]byte(fmt.Sprintf("deleted test #%s from regression database\n", key)))
	}

	return nil
}

// RegressRun runs all the tests in the regression database. The keys argument
// lists specified which entries to test. an empty keys list means that every
// entry should be tested.
func RegressRun(output io.Writer, verbose bool, keys []string) error {
	if output == nil {
		return fmt.Errorf("regression: io.Writer should not be nil")
	}

	dbPth, err := resources.JoinPath(regressionPath, regressionDBFile)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	db, err := database.StartSession(dbPth, database.ActivityReading, initDBSession)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}
	defer db.EndSession(false)

	keys, err = addFailsToKeys(keys)
	if err != nil {
		if errors.Is(err, noPreviousFails) {
			// print message about there being no previous fails after everything else
			defer output.Write([]byte(fmt.Sprintf("no previous fails to re-run\n")))

			// if other keys have been specified allow them to be run. if the
			// list of keys is empty then that is treated as being all keys,
			// which in the context of "FAILS" being a key is probably not what
			// the user wants
			if len(keys) == 0 {
				return nil
			}
		} else {
			return fmt.Errorf("regression: %w", err)
		}
	}

	var successes []string
	var fails []string
	var errors []string

	defer func() {
		output.Write([]byte(fmt.Sprintf("regression tests: %d succeed, %d fail", len(successes), len(fails))))

		if len(errors) > 0 {
			output.Write([]byte(fmt.Sprintf(" [with %d errors]", len(errors))))
		}
		output.Write([]byte("\n"))

		err := saveFails(fails)
		if err != nil {
			output.Write([]byte(fmt.Sprintf("*** error writing fail log: %s\n", err.Error())))
		}
	}()

	// selectKeys() calls this onSelect function for every key entry
	onSelect := func(ent database.Entry, key int) error {
		// database entry should also satisfy Regressor interface
		reg, ok := ent.(Regressor)
		if !ok {
			return fmt.Errorf("regression: database entry does not satisfy Regressor interface")
		}

		// run regress() function with message. message does not have a
		// trailing newline
		msg := fmt.Sprintf("running: %s", reg)
		ok, msg, err := reg.regress(false, output, msg)

		// once regress() has completed we clear the line ready for the
		// completion message
		output.Write([]byte(ansiClearLine))

		// print message depending on result of regress()
		if err != nil {
			errors = append(errors, strconv.Itoa(key))
			fails = append(fails, strconv.Itoa(key))
			output.Write([]byte(fmt.Sprintf("\rfailure: %s\n", reg)))
			if verbose {
				output.Write([]byte(fmt.Sprintf("  ^^ %s\n", err)))
			}
		} else if !ok {
			fails = append(fails, strconv.Itoa(key))
			output.Write([]byte(fmt.Sprintf("\rfailure: %s\n", reg)))
			if verbose && msg != "" {
				output.Write([]byte(fmt.Sprintf("  ^^ %s\n", msg)))
			}
		} else {
			successes = append(successes, strconv.Itoa(key))
			output.Write([]byte(fmt.Sprintf("\rsucceed: %s\n", reg)))
		}

		return nil
	}

	keysInt, err := convertKeys(keys)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	_, err = db.SelectKeys(onSelect, keysInt...)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	return nil
}

// returns true if response from user begins with 'y' or 'Y'.
func confirm(confirmation io.Reader) bool {
	confirm := make([]byte, 32)
	_, err := confirmation.Read(confirm)
	if err != nil {
		return false
	}

	if confirm[0] == 'y' || confirm[0] == 'Y' {
		return true
	}
	return false
}

func convertKeys(keys []string) ([]int, error) {
	keysInt := make([]int, 0, len(keys))
	for i := range keys {
		v, err := strconv.Atoi(keys[i])
		if err != nil {
			return nil, fmt.Errorf("invalid key [%s]", keys[i])
		}
		keysInt = append(keysInt, v)
	}
	sort.Ints(keysInt)
	return slices.Compact(keysInt), nil
}
