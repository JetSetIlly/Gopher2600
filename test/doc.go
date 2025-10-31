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

// Package test contains helper functions for standard Go testing.
//
// The ExpectEquality() is the most basic and probably the most useful function.
// It compares like-typed variables for equality and returns true if they match.
// ExpectInequality() is the inverse function.
//
// The ExpectFailure() and ExpectSuccess() functions test for failure and
// success. These two functions work with bool or error and special handling for
// nil.
//
// All functions a return a boolean to indicate whether the test has passed. This allows
// the user to control larger test procedures that have several stages.
//
// The tags argument for every public function allows the test to be tagged with
// additional information. Each tag will appear at the beginning of any error
// message separated by a colon. For example,
//
//	v := false
//	testFile = "my test data"
//	lineNo = 100
//	test.ExpectSuccess(t, v, testFile, lineNo)
//
// will fail with the following:
//
//	my test data: 100: a success value is expected for type bool
package test
