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

// Package modalflag is a wrapper for the flag package in the Go standard
// library.  It provides a convenient method of handling program modes (and
// sub-modes) and allows different flags for each mode.
//
// At it's simplest it can be used as a replacement for the flag package, with
// some differences. Whereas, with flag.FlagSet you call Parse() with the array of
// strings as the only argument, with modalflag you first NewArgs() with the
// array of arguments and then Parse() with no arguments. For example (note
// that no error handling of the Parse() function is shown here):
//
//	md = Modes{Output: os.Stdout}
//	md.NewArgs(os.Args[1:])
//	_, _ = md.Parse()
//
// The reason for his difference is to allow effective parsing of modes and
// sub-modes. We'll come to program modes in a short while.
//
// In the above example, once the arguments have been parsed, non-flag arguments
// can be retrieved with the RemainingArgs() or GetArg() function. For example,
// handling exactly one argument:
//
//	switch len(md.RemainingArgs()) {
//	case 0:
//		return fmt.Errorf("argument required")
//	case 1:
//		Process(md.GetArg(0))
//	default:
//		return fmt.Errorf("too many arguments")
//	}
//
// Adding flags is similar to the flag package. Adding a boolean flag:
//
// 	verbose := md.AddBool("verbose", false, "print additional log messages")
//
// These flag functions return a pointer to a variable of the specified type. The
// initial value of these variables if the default value, the second argument in
// the function call above. The Parse() function will set these values
// appropriately according what the user has requested, for example:
//
//	if *verbose {
//		fmt.Println(additionalLogMessage)
//	}
//
// The most important difference between the standard flag package and the
// modalflag package is the ability of the latter to handle "modes". In this
// context, a mode is a special command line argument that when specified, puts
// the program into a different mode of operation. The best and most relevant
// example I can think of is the go command. The go command has many different
// modes: build, doc, get, test, etc. Each of these modes are different enough
// to require a different set of flags and expected arguments.
//
// The modalflag package handles sub-modes with the AddSubModes() function.
// This function takes any number of string arguments, each one the name of a
// mode.
//
//	md.AddSubModes("run", "test", "debug")
//
// For simplicity, all sub-mode comparisons are case insensitive.
//
// Subsequent calls to Parse() will then process flags in the normal way but
// unlike the regular flag.Parse() function will check to see if the first
// argument after the flags is one of these modes. If it is, then the
// RemainingArgs() function will return all the arguments after the flags AND
// the mode selector.
//
// So, for example:
//
//	md.Parse()
//	switch md.Mode() {
//		case "RUN":
//			runMode(*verbose)
//		default:
//			fmt.Printf("%s not yet implemented", md.Mode())
//	}
//
// Now that we've decided on what mode we're in, we can again call Parse() to
// process the remaining arguments. This example shows how we can handle return
// state and errors from the Parse() function:
//
//	func runMode(verbose bool) {
//		md.NewMode()
//		md.AddDuration("runtime", time.ParseDuration("10s"), "max run time")
//		p, err := md.Parse
//		switch p {
//			case ParseError:
//				fmt.Println(err)
//				return
//			case ParseHelp:
//				return
//		}
//		doRun(md.RemainingArguments)
//	}
//
// This second call to Parse() will check for any additional flags and any
// further sub-modes (none in this example).
//
// We can chain modes together as deep as we want. For example, the "test" mode
// added above could be divided into several different modes:
//
//	md = Modes{Output: os.Stdout}
//	md.NewArgs(os.Args[1:])
//	md.AddSubModes("run", "test", "debug")
//	_, _ = md.Parse()
//	switch md.Mode() {
//		case "TEST":
//			md.NewMode()
//			md.AddSubModes("A", "B", "C")
//			_, _ = md.Parse()
//			switch md.Mode() {
//				case "A":
//					testA()
//				case "B":
//					testB()
//				case "C":
//					testC()
//			}
//		default:
//			fmt.Printf("%s not yet implemented", md.Mode())
//	}
//
package modalflag
