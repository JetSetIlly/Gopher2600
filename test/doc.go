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

// Package test contains helper functions to remove common boilerplate to make
// testing easier.
//
// The ExpectEquality() is the most basic and probably the most useful function.
// It compares like-typed variables for equality and returns true if they match.
// ExpectInequality() is the inverse function.
//
// The ExpectFailure() and ExpectSuccess() functions test for failure and
// success. These two functions work with bool or error and special handling for
// nil.
//
// ExpectImplements() tests whether an instance implements the specified type.
//
// The Writer type meanwhile, implements the io.Writer interface and should be
// used to capture output. The Writer.Compare() function can then be used to
// test for equality.
package test
