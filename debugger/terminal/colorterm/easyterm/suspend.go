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

// Package easyterm is a wrapper for "github.com/pkg/term/termios". it provides
// some features not present in the third-party package, such as terminal
// geometry, and wraps termios methods in functions with friendlier names
package easyterm

import (
	"os"
	"syscall"
)

// SuspendProcess manually suspends the current process. This is useful if
// terminal is in raw mode and the terminal is given the suspend signal.
func SuspendProcess() {
	p, err := os.FindProcess(os.Getppid())
	if err != nil {
		panic("debugger doesn't seem to have a parent process")
	} else {
		// send TSTP signal to parent proces
		p.Signal(syscall.SIGTSTP)
	}
}
