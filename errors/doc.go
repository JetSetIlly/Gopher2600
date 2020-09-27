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

// Package errors is a helper package for the plain Go language error type. We
// think of these errors as curated errors. External to this package, curated
// errors are referenced as plain errors (ie. they implement the error
// interface).
//
// Internally, errors are thought of as being composed of parts, as described
// by The Go Programming Language (Donovan, Kernighan): "When the error is
// ultimately handled by the program's main function, it should provide a clear
// causal chain from the root of the problem to the overal failure".
//
// The Error() function implementation for curated errors ensures that this
// chain is normalised. Specifically, that the chain does not contain duplicate
// adjacent parts. The practical advantage of this is that it alleviates the
// problem of when and how to wrap errors. For example:
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
//			return errors.Errorf("debugger error: %v", err)
//		}
//		return nil
//	}
//
//	func B() error {
//		err := C()
//		if err != nil {
//			return errors.Errorf("debugger error: %v", err)
//		}
//		return nil
//	}
//
//	func C() error {
//		return errors.Errorf("not yet implemented")
//	}
//
// This will result in the main() function printing an error message. Using the
// curated Error() function, the message will be:
//
//	debugger error: not yet implemented
//
// and not:
//
//	debugger error: debugger error: not yet implemented
//
package errors
