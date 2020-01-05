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

// Package errors is a helper package for the error type. It defines the
// AtariError type, a implementation of error the error interface, that allows
// code to wrap errors around other errors and to allow normalised formatted
// output of error messages.
//
// The most useful feature is deduplication of wrapped errors. This means that
// code does not need to worry about the immediate context of the function
// which creates the error. For instance:
//
//	func main() {
//		err := A()
//		if err != nil {
//			fmt.Println(err)
//		}
//	}
//
//	func A() error {
//		err := B()
//		if err != nil {
//			return errors.New(errors.DebuggerError, err)
//		}
//		return nil
//	}
//
//	func B() error {
//		err := C()
//		if err != nil {
//			return errors.New(errors.DebuggerError, rr)
//		}
//		return nil
//	}
//
//	func C() error {
//		return errors.New(errors.PanicError, "C()", "not yet implemented")
//	}
//
// If we follow the code from main() we can see that first error created is a
// PanicError, wrapped in a DebuggerError, wrapped in a DebuggerError. The
// message for the returned error to main() will be:
//
//	error debugging vcs: panic: C(): not yet implemented
//
// and not
//
//	error debugging vcs: error debugging vcs: panic: C(): not yet implemented
//
// The PanicError, used in the above example, is a special error that should be
// used when something has happened such that the state of the emulation (or
// the tool) can no longer be guaranteed.
//
// Actual panics should only be used when the error is so terrible that there
// is nothing sensible to be done; useful for brute-enforcement of programming
// constraints and in init() functions.
package errors
