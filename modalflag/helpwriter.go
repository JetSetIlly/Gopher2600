package modalflag

import (
	"fmt"
	"io"
	"strings"
)

// helpWriter is used to amend the default output from the flag package
type helpWriter struct {
	// the last []byte sent to the Write() function
	buffer []byte
}

// Clear contents of output buffer
func (hw *helpWriter) Clear() {
	hw.buffer = []byte{}
}

func (hw *helpWriter) Help(output io.Writer, banner string, subModes []string) {
	s := string(hw.buffer)
	helpLines := strings.Split(s, "\n")

	// ouput "no help available" message if there is no flag information and no
	// sub-modes
	if s == "Usage:\n" && len(subModes) == 0 {
		output.Write([]byte("No help available"))
		if banner != "" {
			output.Write([]byte(fmt.Sprintf(" for %s", banner)))
		}
		output.Write([]byte("\n"))
		return
	}

	if banner != "" {
		// supplement default banner with additional string
		output.Write([]byte(fmt.Sprintf("%s for %s mode\n", helpLines[0], banner)))
	} else {
		// there is no banner so just print the default flag package banner
		output.Write([]byte(helpLines[0]))
		output.Write([]byte("\n"))
	}

	// add help message produced by flag package
	if len(helpLines) > 1 {
		s := strings.Join(helpLines[1:], "\n")
		output.Write([]byte(s))
	}

	// add sub-mode information
	if len(subModes) > 0 {

		// add an additional new line if we've already printed flag information
		if len(helpLines) > 2 {
			output.Write([]byte("\n"))
		}

		output.Write([]byte(fmt.Sprintf("  available sub-modes: %s\n", strings.Join(subModes, ", "))))
		output.Write([]byte(fmt.Sprintf("    default: %s\n", subModes[0])))
	}
}

// Write buffers all output
func (hw *helpWriter) Write(p []byte) (n int, err error) {
	hw.buffer = append(hw.buffer, p...)
	return len(p), nil
}
