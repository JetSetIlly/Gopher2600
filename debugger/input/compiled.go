package input

// Commands is the root of the argument "tree"
type Commands map[string]commandArgList

// commandArgList is the list of commandArgList for each command
type commandArgList []commandArg

// maximumLen returns the maximum number of arguments allowed for a given
// command
func (a commandArgList) maximumLen() int {
	if len(a) == 0 {
		return 0
	}
	if a[len(a)-1].typ == argIndeterminate {
		// to indicate indeterminancy, return the maximum value allowed for an integer
		return int(^uint(0) >> 1)
	}
	return len(a)
}

// requiredLen returns the number of arguments required for a given command.
// in other words, the command may allow more but it must have at least the
// returned numnber.
func (a commandArgList) requiredLen() (m int) {
	for i := 0; i < len(a); i++ {
		if !a[i].required {
			return
		}
		m++
	}
	return
}

// argType defines the expected argument type
type argType int

// the possible values for argType
const (
	argKeyword argType = iota
	argFile
	argValue
	argString
	argIndeterminate
	argNode
)

// commandArg specifies the type and properties of an individual argument
type commandArg struct {
	typ      argType
	required bool
	values   interface{}
}
