package debugger

import (
	"gopher2600/errors"
	"gopher2600/gui"
	"io/ioutil"
	"os"
	"strings"
)

type debuggingScript struct {
	scriptFile string
	lines      []string
	nextLine   int
}

func (dbg *Debugger) loadScript(scriptfile string) (*debuggingScript, error) {
	// open script and defer closing
	sf, err := os.Open(scriptfile)
	if err != nil {
		return nil, errors.NewFormattedError(errors.ScriptFileCannotOpen, err)
	}
	defer func() {
		_ = sf.Close()
	}()

	buffer, err := ioutil.ReadAll(sf)
	if err != nil {
		return nil, errors.NewFormattedError(errors.ScriptFileError, err)
	}

	dbs := new(debuggingScript)
	dbs.scriptFile = scriptfile

	// convert buffer to an array of lines
	dbs.lines = strings.Split(string(buffer), "\n")

	return dbs, nil
}

// IsInteractive satisfies the console.UserRead interface
func (dbs *debuggingScript) IsInteractive() bool {
	return false
}

// UserRead implements ui.UserInput interface
func (dbs *debuggingScript) UserRead(buffer []byte, prompt string, _ chan gui.Event, _ func(gui.Event) error) (int, error) {
	if dbs.nextLine > len(dbs.lines)-1 {
		return -1, errors.NewFormattedError(errors.ScriptEnd, dbs.scriptFile)
	}

	l := len(dbs.lines[dbs.nextLine]) + 1
	copy(buffer, []byte(dbs.lines[dbs.nextLine]))
	dbs.nextLine++

	return l, nil
}
