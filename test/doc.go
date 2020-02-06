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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

// Package test contains helper functions to remove common boilerplate to make
// testing easier.
//
// The ExpectedFailure and ExpectedSuccess functions test for failure and
// success under generic conditions. The documenation for those functions
// describe the currently supported types.
//
// It is worth describing how the "Expected" functions handle the nil type
// because it is not obvious. The nil type is considered a success and
// consequently will cause ExpectedFailure to fail and ExpectedSuccess to
// suceed. This may not be how we want to interpret nil in all situations but
// because of how errors usually works (nil to indicate no error) we *need* to
// interpret nil in this way.  If the nil was a value of a nil type we wouldn't
// have this problem incidentally, but that isn't how Go is designed (with good
// reason).
//
// The Writer type meanwhile, implements the io.Writer interface and should be
// used to capture output. The Writer.Compare() function can then be used to
// test for equality.
//
// The Equate() function compares like-typed variables for equality. Some
// types (eg. uint16) can be compared against int for convenience. See Equate()
// documentation for discussion why.
//
// The two "assert thread" functions, AssertMainThread() and
// AssertNonMainThread() will panic if they are not called from, respectively,
// the main thread or from a non-main thread. The package such that these
// assertions are not active unless the "assertions" build tag is specified at
// compile time.
//
// By convention throughout the reset of the project's code, functions should
// be commented with the #mainthread "hashtag" to indicate critical sections
// and calling requirements of the the function. something like
//
//		// MUST ONLY be called from the #mainthread
//
//	or
//
//		// MUST NOT be called from the #mainthread
//
// This makes it easier for the maintainer to spot and search for criticial
// sectoin but such functions should also be supported by calls to
// AssertMainThread() etc.

package test
