// easyterm is a wrapper for "github.com/pkg/term/termios". it provides some
// features not present in the third-party package, such as terminal geometry,
// and wraps termios methods in functions with friendlier names

package easyterm

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/pkg/term/termios"
)

// TermGeometry contains the dimensions of a terminal (usually the output
// terminal)
type TermGeometry struct {
	// characters
	rows uint16
	cols uint16

	// pixels
	x uint16
	y uint16
}

// Terminal is the main container for posix terminals. usually embedded in
// other struct types
type Terminal struct {
	inputFd  uintptr
	outputFd uintptr

	Geometry TermGeometry

	canAttr syscall.Termios
	rawAttr syscall.Termios
}

// Initialise the fields in the Terminal struct
func (pt *Terminal) Initialise(inputFile, outputFile *os.File) error {
	if inputFile != nil {
		pt.inputFd = inputFile.Fd()
	}
	if outputFile != nil {
		pt.outputFd = outputFile.Fd()
	}
	termios.Tcgetattr(pt.inputFd, &pt.canAttr)
	termios.Cfmakecbreak(&pt.rawAttr)
	return nil
}

// UpdateGeometry gets the current dimensions (in characters and pixels) of the
// output terminal
func (pt *Terminal) UpdateGeometry() error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, pt.outputFd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&pt.Geometry)))
	if errno != 0 {
		return fmt.Errorf("error updating terminal geometry information (%d)", errno)
	}
	return nil
}

// CanonicalMode puts terminal into normal, everyday canonical mode
func (pt Terminal) CanonicalMode() {
	termios.Tcsetattr(pt.inputFd, termios.TCIFLUSH, &pt.canAttr)
}

// RawMode puts terminal into raw mode
func (pt Terminal) RawMode() {
	termios.Tcsetattr(pt.inputFd, termios.TCIFLUSH, &pt.rawAttr)
}

// Flush makes sure the terminal's input/output buffers are empty
func (pt Terminal) Flush() error {
	err := termios.Tcflush(pt.inputFd, termios.TCIFLUSH)
	if err != nil {
		return err
	}
	err = termios.Tcflush(pt.outputFd, termios.TCOFLUSH)
	if err != nil {
		return err
	}
	return nil
}
