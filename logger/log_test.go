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
	"testing"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/test"
)

// test central logger and the use of the Tail() function
func TestCentralLogger(t *testing.T) {
	log := logger.NewLogger(100)
	tw := &test.Writer{}

	log.Write(tw)
	test.ExpectEquality(t, tw.Compare(""), true)

	log.Log(logger.Allow, "test", "this is a test")
	log.Write(tw)
	test.ExpectEquality(t, tw.Compare("test: this is a test\n"), true)

	// clear the test.Writer buffer before continuing, makes comparisons easier
	// to manage
	tw.Clear()

	log.Log(logger.Allow, "test2", "this is another test")
	log.Write(tw)
	test.ExpectEquality(t, tw.Compare("test: this is a test\ntest2: this is another test\n"), true)

	// asking for too many entries in a Tail() should be okay
	tw.Clear()
	log.Tail(tw, 100)
	test.ExpectEquality(t, tw.Compare("test: this is a test\ntest2: this is another test\n"), true)

	// asking for exactly the correct number of entries is okay
	tw.Clear()
	log.Tail(tw, 2)
	test.ExpectEquality(t, tw.Compare("test: this is a test\ntest2: this is another test\n"), true)

	// asking for fewer entries is okay too
	tw.Clear()
	log.Tail(tw, 1)
	test.ExpectEquality(t, tw.Compare("test2: this is another test\n"), true)

	// and no entries
	tw.Clear()
	log.Tail(tw, 0)
	test.ExpectEquality(t, tw.Compare(""), true)
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
	tw := &test.Writer{}

	var p prohibitLogging

	for range 100 {
		p.allow = rand.IntN(100)
		log.Clear()
		tw.Clear()
		log.Log(p, "tag", "detail")
		log.Write(tw)
		if p.AllowLogging() {
			test.ExpectEquality(t, tw.Compare("tag: detail\n"), true)
		} else {
			test.ExpectEquality(t, tw.Compare(""), true)
		}
	}
}

// the Log() function explicitly handles error types by using the Error() result
func TestErrorLogging(t *testing.T) {
	log := logger.NewLogger(100)
	tw := &test.Writer{}

	err := errors.New("test error")

	log.Log(logger.Allow, "tag", err)
	log.Write(tw)
	fmt.Println(tw.String())
	test.ExpectEquality(t, tw.Compare("tag: test error\n"), true)

	log.Clear()
	tw.Clear()

	// test "wrapping" of errors using the %v verb
	log.Logf(logger.Allow, "tag", "wrapped: %v", err)
	log.Write(tw)
	fmt.Println(tw.String())
	test.ExpectEquality(t, tw.Compare("tag: wrapped: test error\n"), true)
}

// the Log() function explicitly handles Stringer types
type stringerTest struct{}

func (_ stringerTest) String() string {
	return "stringer test"
}

func TestStringerLogging(t *testing.T) {
	log := logger.NewLogger(100)
	tw := &test.Writer{}

	log.Log(logger.Allow, "tag", stringerTest{})
	log.Write(tw)
	fmt.Println(tw.String())
	test.ExpectEquality(t, tw.Compare("tag: stringer test\n"), true)
}

// for explicitly unsupported types, the Log() function will log the detail
// argument using the %v verb from the fmt package
func TestIntLogging(t *testing.T) {
	log := logger.NewLogger(100)
	tw := &test.Writer{}

	log.Log(logger.Allow, "tag", 100)
	log.Write(tw)
	fmt.Println(tw.String())
	test.ExpectEquality(t, tw.Compare("tag: 100\n"), true)
}
