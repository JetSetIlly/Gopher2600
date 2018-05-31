package ui

// UserInterface defines the user interface operations required by the debugger
type UserInterface interface {
	Initialise() error
	CleanUp()
	RegisterTabCompleter(TabCompleter)
	UserPrint(PrintProfile, string, ...interface{})
	UserRead([]byte, string) (int, error)
}

// UserInterrupt can be returned by UserRead() when user has cause an
// interrupt (ie. CTRL-C)
type UserInterrupt struct {
	Message string
}

// implement Error interface for UserInterrupt
func (intr UserInterrupt) Error() string {
	return intr.Message
}
