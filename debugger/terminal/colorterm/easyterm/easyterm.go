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

// Package easyterm is a wrapper for "github.com/pkg/term/termios". it provides
// some features not present in the third-party package, such as terminal
// geometry, and wraps termios methods in functions with friendlier names
package easyterm

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"unsafe"

	"github.com/pkg/term/termios"
)

// termGeometry contains the dimensions of a terminal (usually the output terminal).
type termGeometry struct {
	rows uint16 // nolint: structcheck
	cols uint16 // nolint: structcheck
	x    uint16 // nolint: structcheck
	y    uint16 // nolint: structcheck
}

// EasyTerm is the main container for posix terminals. usually embedded in
// other struct types.
type EasyTerm struct {
	input  *os.File
	output *os.File

	geometry termGeometry

	canAttr    syscall.Termios
	rawAttr    syscall.Termios
	cbreakAttr syscall.Termios

	// sig/ack channels to control signal handler
	terminateHandlerSig chan bool
	terminateHandlerAck chan bool

	// critical sectioning
	crit sync.Mutex
}

// Initialise the fields in the Terminal struct.
func (et *EasyTerm) Initialise(inputFile, outputFile *os.File) error {
	// not which files we're using for input and output
	if inputFile == nil {
		return fmt.Errorf("easyterm Terminal requires an input file")
	}
	if outputFile == nil {
		return fmt.Errorf("easyterm Terminal requires an output file")
	}

	et.input = inputFile
	et.output = outputFile

	// prepare the attributes for the different terminal modes we'll be using
	err := termios.Tcgetattr(et.input.Fd(), &et.canAttr)
	if err != nil {
		return err
	}

	termios.Cfmakecbreak(&et.cbreakAttr)
	termios.Cfmakeraw(&et.rawAttr)

	// set up sig/ack channels for signal handler
	et.terminateHandlerSig = make(chan bool)
	et.terminateHandlerAck = make(chan bool)

	// kickstart signal handler (it is so cool that this works so easily with
	// go channels)
	go func() {
		sigwinch := make(chan os.Signal, 1)
		signal.Notify(sigwinch, syscall.SIGWINCH)
		defer func() {
			et.terminateHandlerAck <- true
		}()

		for {
			select {
			case <-sigwinch:
				_ = et.UpdateGeometry()
			case <-et.terminateHandlerSig:
				return
			}
		}
	}()

	return nil
}

// CleanUp closes resources created in the Initialise() function.
func (et *EasyTerm) CleanUp() {
	et.crit.Lock()
	defer et.crit.Unlock()

	et.terminateHandlerSig <- true
	<-et.terminateHandlerAck
}

// UpdateGeometry gets the current dimensions (in characters and pixels) of the
// output terminal.
func (et *EasyTerm) UpdateGeometry() error {
	et.crit.Lock()
	defer et.crit.Unlock()

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, et.output.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&et.geometry)))

	if errno != 0 {
		return fmt.Errorf("error updating terminal geometry information (%d)", errno)
	}

	return nil
}

// CanonicalMode puts terminal into normal, everyday canonical mode.
func (et *EasyTerm) CanonicalMode() error {
	et.crit.Lock()
	defer et.crit.Unlock()

	return termios.Tcsetattr(et.input.Fd(), termios.TCIFLUSH, &et.canAttr)
}

// RawMode puts terminal into raw mode.
func (et *EasyTerm) RawMode() error {
	et.crit.Lock()
	defer et.crit.Unlock()

	return termios.Tcsetattr(et.input.Fd(), termios.TCIFLUSH, &et.rawAttr)
}

// CBreakMode puts terminal into cbreak mode.
func (et *EasyTerm) CBreakMode() error {
	et.crit.Lock()
	defer et.crit.Unlock()

	return termios.Tcsetattr(et.input.Fd(), termios.TCIFLUSH, &et.cbreakAttr)
}

// Flush makes sure the terminal's input/output buffers are empty.
func (et *EasyTerm) Flush() error {
	et.crit.Lock()
	defer et.crit.Unlock()

	if err := termios.Tcflush(et.input.Fd(), termios.TCIFLUSH); err != nil {
		return err
	}
	if err := termios.Tcflush(et.output.Fd(), termios.TCOFLUSH); err != nil {
		return err
	}
	return nil
}

// TermPrint writes string to the output file.
func (et *EasyTerm) TermPrint(s string) {
	// no need to take hold of the mutex
	et.output.WriteString(s)
}
