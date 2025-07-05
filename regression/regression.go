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
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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

	// perform the regression test for the regression type
	regress(newRegression bool, messages io.Writer, tag string) error

	// rerun the regression using the current emulation. The returned regressor
	// is a copy of the regressor before it was reduxed. For some regressors it
	// may be the exact same instance
	redux(messages io.Writer, tag string) (Regressor, error)

	// returns true if the regressor is safe to run concurrently
	concurrentSafe() bool
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
		return fmt.Errorf("regression: messages should not be nil")
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
		return fmt.Errorf("regression: messages should not be nil")
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

	err = reg.regress(true, output, fmt.Sprintf("adding: %s", reg))
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	output.Write([]byte(ansiClearLine))
	output.Write([]byte(fmt.Sprintf("\radded: %s\n", reg)))

	return db.Add(reg)
}

// RegressRedux removes and adds an entry using the same parameters.
func RegressRedux(messages io.Writer, confirmation io.Reader, verbose bool, keys []string) error {
	if messages == nil {
		return fmt.Errorf("regression: messages should not be nil")
	}

	if confirmation == nil {
		return fmt.Errorf("regression: confirmation should not be nil")
	}

	// make sure this is what the user wants
	messages.Write([]byte("redux is a dangerous operation. it will rerun all compatible regression entries.\n"))
	messages.Write([]byte("redux? (y/n): "))
	if !confirm(confirmation) {
		return nil
	}
	messages.Write([]byte("sure? (y/n): "))
	if !confirm(confirmation) {
		return nil
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

	// selectKeys() calls this onSelect function for every key entry
	onSelect := func(e database.Entry, key int) error {
		reg, ok := e.(Regressor)
		if !ok {
			messages.Write([]byte("skipping: not a valid regressor"))
			return nil
		}

		old, err := reg.redux(messages, fmt.Sprintf("reduxing: %s", reg))
		if err != nil {
			messages.Write([]byte(fmt.Sprintf("\rfailure: %s\n", reg)))
			if verbose {
				messages.Write([]byte(fmt.Sprintf("  ^^ %s\n", err)))
			}
			return nil
		}

		messages.Write([]byte(ansiClearLine))
		messages.Write([]byte(fmt.Sprintf("\rreduxed: %s\n", reg)))

		err = db.Replace(key, old, reg)
		if err != nil {
			return err
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

// CleanupScript removes orphaned script files from disk. An orphaned file is
// one that exists on disk but has no reference in the regression database
// file.
func RegressCleanup(messages io.Writer, confirmation io.Reader) error {
	if messages == nil {
		return fmt.Errorf("regression: messages should not be nil")
	}

	if confirmation == nil {
		return fmt.Errorf("regression: confirmation should not be nil")
	}

	messages.Write([]byte("cleanup is a dangerous operation. it will delete all orphaned script files.\n"))

	messages.Write([]byte("cleanup? (y/n): "))
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
		messages.Write([]byte(fmt.Sprintf("regression cleanup: %d deleted, %d remain, %d errors\n", numDeleted, numRemain, numErrors)))
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
			messages.Write([]byte(fmt.Sprintf("delete %s? (y/n): ", n)))
			if confirm(confirmation) {
				messages.Write([]byte(ansiClearLine))
				err := os.Remove(n)
				if err != nil {
					messages.Write([]byte(fmt.Sprintf("\rerror deleting: %s (%s)\n", n, err)))
					numErrors++
				} else {
					messages.Write([]byte(fmt.Sprintf("\rdeleted: %s\n", n)))
					numDeleted++
					numRemain--
				}
			} else {
				messages.Write([]byte(ansiClearLine))
				messages.Write([]byte(fmt.Sprintf("\rnot deleting: %s\n", n)))
			}
		} else {
			messages.Write([]byte(fmt.Sprintf("\rkeeping: %s\n", n)))
		}
	}

	return nil
}

// RegressDelete removes a cartridge from the regression db.
func RegressDelete(messages io.Writer, confirmation io.Reader, key string) error {
	if messages == nil {
		return fmt.Errorf("regression: messages should not be nil")
	}

	if confirmation == nil {
		return fmt.Errorf("regression: confirmation should not be nil")
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

	messages.Write([]byte(fmt.Sprintf("%s\ndelete? (y/n): ", ent)))
	if confirm(confirmation) {
		err = db.Delete(v)
		if err != nil {
			return fmt.Errorf("regression: %w", err)
		}
		messages.Write([]byte(fmt.Sprintf("deleted test #%s from regression database\n", key)))
	}

	return nil
}

// RegressRunOptions is passed to RegressRun() to control the function's behaviour
type RegressRunOptions struct {
	Keys        []string
	Verbose     bool
	UseFullPath bool
	Concurrent  bool
}

// RegressRun runs all the tests in the regression database. The keys argument
// lists specified which entries to test. an empty keys list means that every
// entry should be tested.
func RegressRun(messages io.Writer, opts RegressRunOptions) error {
	if messages == nil {
		return fmt.Errorf("regression: messages should not be nil")
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

	var successes []string
	var fails []string

	startTime := time.Now()
	defer func() {
		messages.Write([]byte(fmt.Sprintf("regression tests: %d success, %d fail", len(successes), len(fails))))
		messages.Write([]byte(fmt.Sprintf(" (elapsed %s)\n", time.Since(startTime).Round(time.Second))))
	}()

	// run tests concurrently if possible
	var wg sync.WaitGroup
	var concurrentUnsafe []int

	// function run the actual regression. called from onSelect()
	runTest := func(reg Regressor, key int) {
		// run regress() function with message. message does not have a
		// trailing newline
		err := reg.regress(false, messages, fmt.Sprintf("\r%srunning: %03d %s", ansiClearLine, key, reg))

		// clear line before success/failure message
		messages.Write([]byte(ansiClearLine))

		// print message depending on result of regress()
		if err != nil {
			fails = append(fails, strconv.Itoa(key))
			messages.Write([]byte(fmt.Sprintf("\rfailure: %03d %s\n", key, reg)))
			if opts.Verbose {
				messages.Write([]byte(fmt.Sprintf("  ^^ %s\n", err)))
			}
		} else {
			successes = append(successes, strconv.Itoa(key))
			messages.Write([]byte(fmt.Sprintf("\rsucceed: %03d %s\n", key, reg)))
		}
	}

	// selectKeys() calls this onSelect function for every key entry
	onSelect := func(e database.Entry, key int) error {
		// database entry should also satisfy Regressor interface
		reg, ok := e.(Regressor)
		if !ok {
			return fmt.Errorf("database entry does not satisfy Regressor interface")
		}
		if !opts.Concurrent || !reg.concurrentSafe() {
			concurrentUnsafe = append(concurrentUnsafe, key)
			return nil
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			runTest(reg, key)
		}()

		return nil
	}

	keysInt, err := convertKeys(opts.Keys)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	_, err = db.SelectKeys(onSelect, keysInt...)
	if err != nil {
		return fmt.Errorf("regression: %w", err)
	}

	// wait for all concurrent tests to complete
	wg.Wait()

	// run tests that can not be run concurrently. this uses a simpled onSelect() function
	onSelect = func(e database.Entry, key int) error {
		reg, ok := e.(Regressor)
		if !ok {
			return fmt.Errorf("database entry does not satisfy Regressor interface")
		}
		runTest(reg, key)
		return nil
	}

	// check that concurrent unsafe list has entries otherwise, the SelectKeys()
	// function will match every key
	if len(concurrentUnsafe) > 0 {
		_, err = db.SelectKeys(onSelect, concurrentUnsafe...)
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
