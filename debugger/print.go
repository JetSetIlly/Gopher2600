package debugger

import (
	"gopher2600/debugger/ui"
	"strings"
)

// wrapper function for UserPrint(). useful for  normalising the input string
// before passing to the real UserPrint. it also allows us to easily obey
// directives such as the silent directive without passing the burden onto UI
// implementors
func (dbg *Debugger) print(pp ui.PrintProfile, s string, a ...interface{}) {
	if dbg.uiSilent && pp != ui.Error {
		return
	}

	// trim *all* trailing newlines - UserPrint() will add newlines if required
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return
	}

	dbg.ui.UserPrint(pp, s, a...)
}
