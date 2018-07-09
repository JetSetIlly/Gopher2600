// easyterm is a wrapper for "github.com/pkg/term/termios". it provides some
// features not present in the third-party package, such as terminal geometry,
// and wraps termios methods in functions with friendlier names

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
	input  *os.File
	output *os.File

	Geometry TermGeometry

	canAttr    syscall.Termios
	rawAttr    syscall.Termios
	cbreakAttr syscall.Termios

	// sig/ack channels to control signal handler
	terminateHandlerSig chan bool
	terminateHandlerAck chan bool

	// public functions that are  called from the signal handler are prefaced
	// with (to prevent race conditions, or worse):
	// 		pt.mu.Lock()
	// 		defer pt.mu.Unlock()
	mu sync.Mutex
}

// Initialise the fields in the Terminal struct
func (pt *Terminal) Initialise(inputFile, outputFile *os.File) error {
	// not which files we're using for input and output
	if inputFile == nil {
		return fmt.Errorf("easyterm Terminal requires an input file")
	}
	if outputFile == nil {
		return fmt.Errorf("easyterm Terminal requires an output file")
	}

	pt.input = inputFile
	pt.output = outputFile

	// prepare the attributes for the different terminal modes we'll be using
	termios.Tcgetattr(pt.input.Fd(), &pt.canAttr)
	termios.Cfmakecbreak(&pt.cbreakAttr)
	termios.Cfmakeraw(&pt.rawAttr)

	// set up sig/ack channels for signal handler
	pt.terminateHandlerSig = make(chan bool)
	pt.terminateHandlerAck = make(chan bool)

	// kickstart signal handler (it is so cool that this works so easily with
	// go channels)
	go func() {
		sigwinch := make(chan os.Signal, 1)
		signal.Notify(sigwinch, syscall.SIGWINCH)
		defer func() {
			pt.terminateHandlerAck <- true
		}()

		for {
			select {
			case <-sigwinch:
				_ = pt.UpdateGeometry()
			case <-pt.terminateHandlerSig:
				return
			}
		}
	}()

	return nil
}

// CleanUp closes resources created in the Initialise() function
func (pt *Terminal) CleanUp() {
	pt.terminateHandlerSig <- true
	<-pt.terminateHandlerAck
}

// Print writes the formatted string to the output file
// TODO: expand the functionality of easyterm Print()
func (pt *Terminal) Print(s string, a ...interface{}) {
	pt.output.WriteString(fmt.Sprintf(s, a...))
	pt.output.Sync()
}

// UpdateGeometry gets the current dimensions (in characters and pixels) of the
// output terminal
func (pt *Terminal) UpdateGeometry() error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, pt.output.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&pt.Geometry)))
	if errno != 0 {
		return fmt.Errorf("error updating terminal geometry information (%d)", errno)
	}
	return nil
}

// CanonicalMode puts terminal into normal, everyday canonical mode
func (pt *Terminal) CanonicalMode() {
	termios.Tcsetattr(pt.input.Fd(), termios.TCIFLUSH, &pt.canAttr)
}

// RawMode puts terminal into raw mode
func (pt *Terminal) RawMode() {
	termios.Tcsetattr(pt.input.Fd(), termios.TCIFLUSH, &pt.rawAttr)
}

// CBreakMode puts terminal into cbreak mode
func (pt *Terminal) CBreakMode() {
	termios.Tcsetattr(pt.input.Fd(), termios.TCIFLUSH, &pt.cbreakAttr)
}

// Flush makes sure the terminal's input/output buffers are empty
func (pt *Terminal) Flush() error {
	if err := termios.Tcflush(pt.input.Fd(), termios.TCIFLUSH); err != nil {
		return err
	}
	if err := termios.Tcflush(pt.output.Fd(), termios.TCOFLUSH); err != nil {
		return err
	}
	return nil
}
