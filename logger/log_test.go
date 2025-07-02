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

package logger_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/test"
)

// test central logger and the use of the Tail() function
func TestCentralLogger(t *testing.T) {
	log := logger.NewLogger(100)
	w := &strings.Builder{}

	log.Write(w)
	test.ExpectEquality(t, w.String(), "")

	log.Log(logger.Allow, "test", "this is a test")
	log.Write(w)
	test.ExpectEquality(t, w.String(), "test: this is a test\n")

	// clear the test.Writer buffer before continuing, makes comparisons easier
	// to manage
	w.Reset()

	log.Log(logger.Allow, "test2", "this is another test")
	log.Write(w)
	test.ExpectEquality(t, w.String(), "test: this is a test\ntest2: this is another test\n")

	// asking for too many entries in a Tail() should be okay
	w.Reset()
	log.Tail(w, 100)
	test.ExpectEquality(t, w.String(), "test: this is a test\ntest2: this is another test\n")

	// asking for exactly the correct number of entries is okay
	w.Reset()
	log.Tail(w, 2)
	test.ExpectEquality(t, w.String(), "test: this is a test\ntest2: this is another test\n")

	// asking for fewer entries is okay too
	w.Reset()
	log.Tail(w, 1)
	test.ExpectEquality(t, w.String(), "test2: this is another test\n")

	// and no entries
	w.Reset()
	log.Tail(w, 0)
	test.ExpectEquality(t, w.String(), "")
}

// test permissions by randomising whether logging is allowed or not. there's no
// need to do the randomisation but it's as good a demonstration as anything
// else I can think of
type prohibitLogging struct {
	allow int
}

func (p prohibitLogging) AllowLogging() bool {
	return p.allow > 50
}

func TestPermissions(t *testing.T) {
	log := logger.NewLogger(100)
	w := &strings.Builder{}

	var p prohibitLogging

	for range 100 {
		p.allow = rand.IntN(100)
		log.Clear()
		w.Reset()
		log.Log(p, "tag", "detail")
		log.Write(w)
		if p.AllowLogging() {
			test.ExpectEquality(t, w.String(), "tag: detail\n")
		} else {
			test.ExpectEquality(t, w.String(), "")
		}
	}
}

// the Log() function explicitly handles error types by using the Error() result
func TestErrorLogging(t *testing.T) {
	log := logger.NewLogger(100)
	w := &strings.Builder{}

	err := errors.New("test error")

	log.Log(logger.Allow, "tag", err)
	log.Write(w)
	fmt.Println(w.String())
	test.ExpectEquality(t, w.String(), "tag: test error\n")

	log.Clear()
	w.Reset()

	// test "wrapping" of errors using the %v verb
	log.Logf(logger.Allow, "tag", "wrapped: %v", err)
	log.Write(w)
	fmt.Println(w.String())
	test.ExpectEquality(t, w.String(), "tag: wrapped: test error\n")
}

// the Log() function explicitly handles Stringer types
type stringerTest struct{}

func (_ stringerTest) String() string {
	return "stringer test"
}

func TestStringerLogging(t *testing.T) {
	log := logger.NewLogger(100)
	w := &strings.Builder{}

	log.Log(logger.Allow, "tag", stringerTest{})
	log.Write(w)
	test.ExpectEquality(t, w.String(), "tag: stringer test\n")
}

// for explicitly unsupported types, the Log() function will log the detail
// argument using the %v verb from the fmt package
func TestIntLogging(t *testing.T) {
	log := logger.NewLogger(100)
	w := &strings.Builder{}

	log.Log(logger.Allow, "tag", 100)
	log.Write(w)
	test.ExpectEquality(t, w.String(), "tag: 100\n")
}
