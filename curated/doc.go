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

// Package curated is a helper package for the plain Go language error type.
// Curated errors implement the error interface.
//
// Curated errors are created with the Errorf() function. This is similar to
// the Errorf() function in the fmt package. It takes a formatting pattern,
// placeholder values and returns an error.
//
// The Is() function can be used to check whether an error was created by the
// (Errorf() function). The Errorf() pattern is used to differntiate curated
// errors. For example:
//
//	a := 10
//	e := curated.Errorf("error: value = %d", a)
//
//	if curated.Is(e, "error: value = %d") {
//		fmt.Println("true")
//	}
//
// The Has() function is similar but checks if a pattern occurs somewhere in
// the error chain.
//
//	a := 10
//	e := curated.Errorf("error: value = %d", a)
//	f := curated.Errorf("fatal: %v", e)
//
//	if curated.Has(f, "error: value = %d") {
//		fmt.Println("true")
//	}
//
//	if curated.Is(f, "error: value = %d") {
//		fmt.Println("true")
//	}
//
// Note that in this example, the call to Is() fails will not print 'true'
// because error f does not match that pattern - it is "wrapped" inside the
// pattern "fatal: %v".
//
// The IsAny() function answers whether the error was created by curated.Errorf().
// Put another way, it returns true if the error is 'curated' and false if the
// error is 'uncurated'. Alternatively, we can think of the difference as being
// 'expected' and 'unexpected' depending on how we choose to handle the result
// of the function call.
//
// The Error() function implementation for curated errors ensures that the
// error chain is normalised. Specifically, that the chain does not contain
// duplicate adjacent parts. The practical advantage of this is that it
// alleviates the problem of when and how to wrap curated. For example:
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
//			return curated.Errorf("error: %v", err)
//		}
//		return nil
//	}
//
//	func B() error {
//		err := C()
//		if err != nil {
//			return curated.Errorf("error: %v", err)
//		}
//		return nil
//	}
//
//	func C() error {
//		return curated.Errorf("not yet implemented")
//	}
//
// This will result in the main() function printing an error message. Using the
// curated Error() function, the message will be:
//
//	error: not yet implemented
//
// and not:
//
//	error: error: not yet implemented
//
// For the purposes of this package we think of chains as being composed of
// parts separted by the sub-string ': ' as suggested on p239 of "The Go
// Programming Language" (Donovan, Kernighan). For example:
//
//	part 1: part 2: part 3
//
// There is no special provision for sentinal errors in the curated package but
// they are achievable in practice through the use of the Is() and Has()
// functions. Sentinal pattern should be stored as a const string, suitably
// named and commented. A Sentinal type may be introduced in the future.
package curated
